package everhour

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/jason0x43/go-toggl"
	"gitlab.com/leucos/toji/api/everhour"
	"gitlab.com/leucos/toji/drivers"
	"gitlab.com/leucos/toji/internal/config"
)

type EverHour struct {
	from time.Time
	to   time.Time
	base *drivers.BaseOptions
}

func New(from, to time.Time, opts ...drivers.OptionFunc) (EverHour, error) {
	e := EverHour{
		from: from,
		to:   to,
		base: &drivers.BaseOptions{},
	}

	for _, opts := range opts {
		opts(e.base)
	}
	return e, nil
}

func (e EverHour) Sync(c chan drivers.SyncedEntry) {
	err := e.doSync(c)
	if err != nil {
		slog.Error("unrecoverable error in Jira.Sync", "err", err)
	}
	close(c)
}

func (e EverHour) doSync(c chan drivers.SyncedEntry) error {
	slog.Debug("everhour sync mode", "from", e.from, "to", e.to)

	session := toggl.OpenSession(config.Current.Get("toggl.token"))
	entries, err := session.GetTimeEntries(e.from, e.to)

	if err != nil {
		return fmt.Errorf("unable to fetch Toggl entries: %v. Is your token valid ?", err)
	}

	currentDate := e.from.AddDate(-1, 0, 0).Format("Mon 2006/01/02")

	// firstChange := time.Now()
	// alreadyExistEntries := 0

	projects := []everhour.Project{}
	for _, entry := range projects {
		fmt.Printf("Project: %s\n", entry.Name)
		return nil
	}

	for _, entry := range entries {
		textDate := entry.Start.Format("Mon 2006/01/02")
		if textDate != currentDate {
			// fmt.Printf("\n%s\n==============\n", textDate)
			currentDate = textDate
			// currentProject = ""
		}

		// Project holds the Jira ticket ID (e.g. XYZ-123)
		// project := extractTicketFromString(e.Description)

		// fmt.Printf("")
		// if project == "" {
		// 	slog.Debug("no jira ticket found on entry", "desc", e.Description)
		// 	continue
		// }

		// if we have project filters, check if we have a match
		// if len(projectList) > 0 {
		// 	projectSlug := strings.Split(project, "-")
		// 	if !isInSlice(projectSlug[0], projectList) {
		// 		slog.Debug("skipping project since it is not included", "project", projectSlug[0])
		// 		continue
		// 	}
		// }

		// if entry.StopTime().IsZero() {
		// 	slog.Debug("skipping currently running time entry", "ticket", project)
		// 	continue
		// }

		// if e.base.Issues != nil && !isInSlice(project, e.base.Issues) {
		// 	// fmt.Printf("    skipping time entry (not selected)\n")
		// 	slog.Debug("skipping issue since it is not selected", "issue", project)
		// 	continue
		// }

		// changed, err := entry.updateJiraTracking(project, entry)
		// if err != nil {
		// 	slog.Error("unable to sync issue", "issue", project, "err", err)
		// 	// fmt.Fprintf(os.Stderr, "unable to sync with issue %s: %v", project, err)
		// 	continue
		// }
		// // Keep track of first change date
		// if changed && firstChange.After(*entry.Start) {
		// 	firstChange = *entry.Start
		// }

		// Keep track of how many entries already exist
		// if !changed {
		// 	alreadyExistEntries++
		// } else {
		// 	c <- drivers.SyncedEntry{
		// 		Date:     entry.Start.Format("2006/01/02 Mon"),
		// 		Duration: entry.Duration,
		// 		Type:     drivers.Simple,
		// 		Message:  fmt.Sprintf("[%s] inserted %d from Toggl entry %d to %s's worklog entry", entry.Start.Format("15:04"), entry.Duration, entry.ID, project),
		// 	}
		// }
	}
	return nil
}

func (e EverHour) Rollup(c chan drivers.SyncedEntry) {
}
