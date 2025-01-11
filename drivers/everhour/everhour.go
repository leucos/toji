package everhour

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/leucos/go-toggl"
	"gitlab.com/leucos/toji/api/everhour"
	"gitlab.com/leucos/toji/drivers"
	"gitlab.com/leucos/toji/internal/config"
	t "gitlab.com/leucos/toji/internal/toggl"
)

type EverHour struct {
	from   time.Time
	to     time.Time
	client *everhour.Client
	base   *drivers.BaseOptions
}

func New(from, to time.Time, token string, opts ...drivers.OptionFunc) (EverHour, error) {
	e := EverHour{
		from: from,
		to:   to,
		base: &drivers.BaseOptions{},
	}

	client := everhour.New(token)
	e.client = client

	for _, opts := range opts {
		opts(e.base)
	}
	return e, nil
}

func (e EverHour) Sync(c chan drivers.SyncedEntry) {
	err := e.doSync(c)
	if err != nil {
		slog.Error("unrecoverable error in EverHour.Sync", "err", err)
	}
	close(c)
}

// doSync fetches Toggl entries and syncs them with Everhour
// Logic:
// - fetch all Toggl entries between requested dates
// - for each entry:
//   - extract project name from entry
//   - see if project name matches a config mapping
//   - if so, find the corresponding Everhour project & task

func (e EverHour) doSync(c chan drivers.SyncedEntry) error {
	type ehProjectMapper struct {
		name    string
		project *everhour.Project
	}

	var (
		ok             bool
		alreadyPresent bool
		ehProject      ehProjectMapper
		everHourMapper map[string]ehProjectMapper
	)

	everHourMapper = make(map[string]ehProjectMapper)

	// Get the mapping between Toggl and Everhour projects
	{
		// get the config mapping between toggl and EH
		mapping := config.Current.GetMapString("everhour.mappings")

		if len(mapping) == 0 {
			return fmt.Errorf("no mapping found in config")
		}

		// copy mapping to projectMapper
		for k, v := range mapping {
			slog.Debug("adding project mapping", "toggl", k, "everhour", v)
			everHourMapper[k] = ehProjectMapper{name: v}
		}
	}

	slog.Debug("everhour sync mode", "from", e.from, "to", e.to)

	session := toggl.OpenSession(config.Current.Get("toggl.token"))
	togglEntries, err := session.GetTimeEntries(e.from, e.to)
	if err != nil {
		return fmt.Errorf("unable to fetch Toggl entries: %v. Is your token valid ?", err)
	}

	currentDate := e.from.AddDate(-1, 0, 0).Format("Mon 2006/01/02")

	for _, entry := range togglEntries {

		textDate := entry.Start.Format("Mon 2006/01/02")
		if textDate != currentDate {
			currentDate = textDate
		}

		if entry.Pid == nil {
			continue
		}

		if e.base.Issues != nil {
			if !t.CheckIssueMatches(e.base.Issues, entry.Description) {
				//!isInSlice(project, j.base.Issues) {
				slog.Debug("skipping issue since it is not selected", "issue", entry.Description)
				continue
			}
			slog.Debug("including issue since it is selected", "issue", entry.Description)
		}

		// Extract Toggl project name from entry
		name, err := t.GetProjectNameFromID(session, config.Current.GetInt("toggl.workspace"), *entry.Pid)
		if err != nil {
			return err
		}
		slog.Debug("got project name from id", "id", *entry.Pid, "name", name)

		// Find the corresponding Everhour project & task
		if ehProject, ok = everHourMapper[name]; !ok {
			// We did not find a matching pair
			// Skip this Toggl entry
			continue
		}

		// fmt.Printf("\nEntry desc: %s\n", entry.Description)
		// fmt.Printf("\tproject name: %s\n", name)
		// fmt.Printf("\tstart: %s\n", entry.Start)
		// fmt.Printf("\tmappedProject : %d (%s)\n", entry.Pid, ehProject.name)

		slog.Debug("found toggl/everhour mapping",
			"toggl", name, "everhour", ehProject.name,
			"toggl.ID", entry.ID,
			"toggl.Description", entry.Description,
			"toggl.Start", entry.Start,
		)

		// Find associated project in everhour if needed
		// This lazy loads the mappedProject struct and limits EH API calls
		if ehProject.project == nil {
			projects, err := e.client.GetAllProjects()
			if err != nil {
				return err
			}
			for _, prj := range projects {
				if prj.Name == ehProject.name {
					ehProject.project = &prj
					break
				}
			}
		}

		// Find associated task in everhour
		task, err := e.client.GetTaskByName(entry.Description, ehProject.project.ID)
		if err != nil {
			return err
		}
		if task == nil {
			slog.Warn("everhour task not found", "name", entry.Description)
			continue
		}

		slog.Debug("everhour task found", "name", task.Name, "id", task.ID)

		// Get time entries for task
		entries, err := e.client.GetTaskTimeRecords(task.ID)
		if err != nil {
			continue
		}

		for _, e := range entries {
			slog.Debug("everhour entry", "date", e.Date, "time", e.Time, "comment", e.Comment)
			// Check if toggl entry s present in comments
			if strings.Contains(e.Comment, fmt.Sprintf("Toggl entry %d", entry.ID)) {
				slog.Debug("toggl time entry already present in everhour task", "date", e.Date, "time", e.Time, "comment", e.Comment)
				alreadyPresent = true
				break
			}
		}

		if alreadyPresent {
			alreadyPresent = false
			continue
		}

		// Round if configuration says so
		round := int64(config.Current.GetInt("toggl.rounding")) * 60
		if round > 0 {
			slog.Debug("rounding is set", "rounding", round)
			duration := entry.Duration
			// check remainder
			remainder := duration % round
			if remainder != 0 {
				rounded := duration + (round - remainder)
				if e.base.DryRun {
					slog.Info("dry-run: would have rounded toggl entry", "entry.ID", entry.ID, "entry.Description", entry.Description, "configured rounding seconds", round, "initial duration", duration, "rounded duration", rounded)
				} else {
					err = entry.SetDuration(rounded)
					if err != nil {
						slog.Error("unable to change toggl entry duration", "err", err, "entry.ID", entry.ID, "entry.Description", entry.Description)
					}
					entry, err = session.UpdateTimeEntry(entry)
					if err != nil {
						slog.Error("unable to update toggl entry", "err", err, "entry.ID", entry.ID, "entry.Description", entry.Description)
					} else {
						slog.Debug("rounded toggl entry", "entry.ID", entry.ID, "entry.Description", entry.Description, "configured rounding", round/60, "initial duration", duration, "rounded duration", rounded)
					}
				}
			}
		}

		if e.base.DryRun {
			slog.Info("dry-run: would have synced entry", "entry.ID", entry.ID, "date", entry.Start.Format("2006/01/02 Mon"), "duration", entry.Duration, "task", task.Name)
			continue
		}
		// Create time entry
		err = e.client.AddTime(
			task.ID,
			entry.Start,
			entry.Duration,
			fmt.Sprintf("Toggl entry %d", entry.ID),
		)

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		slog.Info("synced entry", "date", entry.Start.Format("2006/01/02 Mon"), "duration", entry.Duration, "task", task.Name)

		c <- drivers.SyncedEntry{
			Date:     entry.Start.Format("2006/01/02 Mon"),
			Duration: entry.Duration,
			Type:     drivers.Simple,
			Message:  fmt.Sprintf("[%s] inserted %d from Toggl entry %d to %q worklog entry", entry.Start.Format("15:04"), entry.Duration, entry.ID, task.Name),
		}
	}

	// loop over Toggl resources types and show stats
	session.EnableLog(slog.Default())
	session.ShowStats(config.Current.GetInt("toggl.workspace"))
	return nil
}

func (e EverHour) Rollup(c chan drivers.SyncedEntry) {

}
