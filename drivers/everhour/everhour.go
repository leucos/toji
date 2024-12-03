package everhour

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jason0x43/go-toggl"
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

		// Extract Toggl project name from entry
		name, err := t.GetProjectNameFromID(session, config.Current.GetInt("toggl.workspace"), *entry.Pid)
		if err != nil {
			return err
		}

		// Find the corresponding Everhour project & task
		if ehProject, ok = everHourMapper[name]; !ok {
			// We did not find a matching pair
			// Skip this Toggl entry
			continue
		}

		fmt.Printf("\nEntry desc: %s\n", entry.Description)
		fmt.Printf("\tproject name: %s\n", name)
		fmt.Printf("\tstart: %s\n", entry.Start)
		fmt.Printf("\tmappedProject : %d (%s)\n", entry.Pid, ehProject)

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
			fmt.Printf("Task not found: %s\n", entry.Description)
			continue
		}

		fmt.Printf("\tTask: %s (%s)\n", task.Name, task.ID)

		// Get time entries for task
		entries, err := e.client.GetTaskTimeRecords(task.ID)
		if err != nil {
			continue
		}

		for _, e := range entries {
			fmt.Printf("\t\t%s: %d (%s)\n", e.Date, e.Time, e.Comment)
			// Check if toggl entry s present in comments
			if strings.Contains(e.Comment, fmt.Sprintf("Toggl entry %d", entry.ID)) {
				fmt.Printf("\t\t\talready present\n")
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
			fmt.Printf("\t\trounding to %dm requested\n", round)
			duration := entry.Duration
			// check remainder
			remainder := duration % round
			if remainder != 0 {
				rounded := duration + (round - remainder)
				err = entry.SetDuration(rounded)
				if err != nil {
					slog.Error("unable to change toggl entry duration", "err", err, "entry.ID", entry.ID, "entry.Description", entry.Description)
				}
				entry, err = session.UpdateTimeEntry(entry)
				if err != nil {
					slog.Error("unable to update toggl entry", "err", err, "entry.ID", entry.ID, "entry.Description", entry.Description)
				} else {
					slog.Debug("rounded toggl entry", "entry.ID", entry.ID, "entry.Description", entry.Description, "duration", duration, "rounded", rounded)
					fmt.Printf("\t\tRounded toggl entry %d %s from %d to %d\n", entry.ID, entry.Description, duration, rounded)
				}
			}
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

		c <- drivers.SyncedEntry{
			Date:     entry.Start.Format("2006/01/02 Mon"),
			Duration: entry.Duration,
			Type:     drivers.Simple,
			Message:  fmt.Sprintf("[%s] inserted %d from Toggl entry %d to %q worklog entry", entry.Start.Format("15:04"), entry.Duration, entry.ID, task.Name),
		}
	}

	return nil
}

func (e EverHour) Rollup(c chan drivers.SyncedEntry) {
}
