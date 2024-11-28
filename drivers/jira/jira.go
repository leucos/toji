package jira

import (
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/jason0x43/go-toggl"
	"gitlab.com/leucos/toji/drivers"
	"gitlab.com/leucos/toji/internal/config"
)

type Jira struct {
	from time.Time
	to   time.Time
	base *drivers.BaseOptions
}

func New(from, to time.Time, opts ...drivers.OptionFunc) (Jira, error) {
	j := Jira{
		from: from,
		to:   to,
		base: &drivers.BaseOptions{},
	}

	for _, opts := range opts {
		opts(j.base)
	}
	return j, nil
}

func (j Jira) Sync(c chan drivers.SyncedEntry) {
	err := j.doSync(c)
	if err != nil {
		slog.Error("unrecoverable error in Jira.Sync", "err", err)
	}
	close(c)
}

func (j Jira) Rollup(c chan drivers.SyncedEntry) {
	err := j.doRollup(c)
	if err != nil {
		slog.Error("unrecoverable error in Jira.Rollup", "err", err)
	}
	close(c)
}

func (j Jira) doRollup(c chan drivers.SyncedEntry) error {
	todayStart := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)

	if j.to.After(todayStart) {
		j.to = todayStart.Add(-1 * time.Second)
	}

	projectList := []string{}
	// if we have filters, prepare a string slice
	if config.Current.Get("jira.projects") != "" {
		projectList = strings.Split(config.Current.Get("jira.projects"), ",")
	}

	slog.Debug("rollup mode", "from", j.from, "to", j.to)
	// fmt.Printf("\nRolling up toggl entries between %s and %s\n", j.from, j.to)

	session := toggl.OpenSession(config.Current.Get("toggl.token"))
	entries, err := session.GetTimeEntries(j.from, j.to)

	if err != nil {
		return fmt.Errorf("unable to fetch Toggl entries: %v. Is your token valid ?", err)
	}

	currentDate := j.from.AddDate(-1, 0, 0).Format("Mon 2006/01/02")
	// currentProject := ""

	// firstChange := time.Now()
	// alreadyExistEntries := 0

	// first key is date "Mon 2006/01/02"
	// second key is issue ID
	// value is a struct containing the cumulated seconds for the
	// issue and a description
	type singleRollup struct {
		duration    int64
		description string
	}
	dailyRollups := map[string]map[string]*singleRollup{}

	for _, e := range entries {
		textDate := e.Start.Format("2006/01/02 Mon")
		if textDate != currentDate {
			slog.Debug("checking new date", "date", textDate)
			// fmt.Printf("\n%s\n==============\n", textDate)
			currentDate = textDate
			// currentProject = ""
			// fmt.Printf("creating entry for %s\n", textDate)
			dailyRollups[textDate] = make(map[string]*singleRollup)
			currentDate = textDate
		}

		// Project holds the Jira ticket ID (e.g. XYZ-123)
		project := extractTicketFromString(e.Description)

		// fmt.Printf("")
		if project == "" {
			slog.Debug("no jira ticket found on entry", "desc", e.Description)
			continue
		}

		// if we have project filters, check if we have a match
		if len(projectList) > 0 {
			projectSlug := strings.Split(project, "-")
			if !isInSlice(projectSlug[0], projectList) {
				slog.Debug("skipping project since it is not included", "project", projectSlug[0])
				continue
			}
		}

		// Only redisplay project description if the project is not the same as
		// previous iteration
		// if project != currentProject {
		// 	fmt.Printf("\n  %s (%s/browse/%s)\n", e.Description, config.Current.Get("jira.url"), project)
		// 	currentProject = project
		// }

		if e.StopTime().IsZero() {
			slog.Debug("skipping currently running time entry", "ticket", project)
			continue
		}

		if j.base.Issues != nil && !isInSlice(project, j.base.Issues) {
			slog.Debug("skipping issue since it is not selected", "issue", project)
			continue
		}

		slog.Debug("adding duration to project", "duration", e.Duration, "project", project)
		explodedIssue := strings.Split(e.Description, " ")
		if dailyRollups[textDate][explodedIssue[0]] == nil {
			dailyRollups[textDate][explodedIssue[0]] = &singleRollup{duration: e.Duration, description: strings.Join(explodedIssue[1:], " ")}
			slog.Debug("setting description", "final", explodedIssue[1:], "source", e.Description)
		} else {
			dailyRollups[textDate][explodedIssue[0]].duration += e.Duration
		}
	}

	// adjust rounding if needed
	if j.base.Rounding != 0 {
		for day, pmap := range dailyRollups {
			for issue, srp := range pmap {
				// *60 is needed since rounding is expressed as minutes
				// fmt.Printf("day: %s issue: %s\n", day, issue)
				dailyRollups[day][issue].duration += int64(j.base.Rounding*60) - srp.duration%int64(j.base.Rounding*60)
			}
		}
	}

	// fmt.Printf("Rollup mode\n==============\n\n")

	var keys []string
	for key := range dailyRollups {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, k := range keys {
		// fmt.Printf("%s\n--------------\n", k)
		for issueID, srp := range dailyRollups[k] {
			slog.Debug("processing rollup", "datekey", k, "issue", issueID, "description", srp.description, "duration", srp.duration)
			// fmt.Printf("k: %s, issue: %s, description: %s, dur %d\n", k, issueID, srp.description, srp.duration)
			created, err := j.updateJiraRollup(k, issueID, srp.description, srp.duration)
			if err != nil {
				slog.Error("unable to sync issue", "datekey", k, "issue", issueID, "err", err)
				continue
			}

			if created {
				c <- drivers.SyncedEntry{
					Date:     k,
					Duration: srp.duration,
					Type:     drivers.Rollup,
					Message:  fmt.Sprintf("inserted %d seconds from Toggl entries to %s's worklog entry (key %q)", srp.duration, issueID, k),
				}
			}
		}
		// fmt.Println()
	}

	return nil
}

func (j Jira) doSync(c chan drivers.SyncedEntry) error {
	projectList := []string{}
	// if we have filters, prepare a string slice
	if config.Current.Get("jira.projects") != "" {
		projectList = strings.Split(config.Current.Get("jira.projects"), ",")
	}

	slog.Debug("sync mode", "from", j.from, "to", j.to)
	// fmt.Printf("\nSyncing toggl entries between %s and %s\n", j.from, j.to)

	session := toggl.OpenSession(config.Current.Get("toggl.token"))
	entries, err := session.GetTimeEntries(j.from, j.to)

	if err != nil {
		return fmt.Errorf("unable to fetch Toggl entries: %v. Is your token valid ?", err)
	}

	currentDate := j.from.AddDate(-1, 0, 0).Format("Mon 2006/01/02")
	// currentProject := ""

	firstChange := time.Now()
	alreadyExistEntries := 0

	for _, e := range entries {
		textDate := e.Start.Format("Mon 2006/01/02")
		if textDate != currentDate {
			// fmt.Printf("\n%s\n==============\n", textDate)
			currentDate = textDate
			// currentProject = ""
		}

		// Project holds the Jira ticket ID (e.g. XYZ-123)
		project := extractTicketFromString(e.Description)

		fmt.Printf("")
		if project == "" {
			slog.Debug("no jira ticket found on entry", "desc", e.Description)
			continue
		}

		// if we have project filters, check if we have a match
		if len(projectList) > 0 {
			projectSlug := strings.Split(project, "-")
			if !isInSlice(projectSlug[0], projectList) {
				slog.Debug("skipping project since it is not included", "project", projectSlug[0])
				// fmt.Printf("    skipping since project not included for entry %s\n", project)
				continue
			}
		}

		// Only redisplay project description if the project is not the same as
		// previous iteration
		// if project != currentProject {
		// 	fmt.Printf("\n  %s (%s/browse/%s)\n", e.Description, config.Current.Get("jira.url"), project)
		// 	currentProject = project
		// }

		if e.StopTime().IsZero() {
			// fmt.Printf("    skipping currently running time entry for %s\n", project)
			slog.Debug("skipping currently running time entry", "ticket", project)
			continue
		}

		if j.base.Issues != nil && !isInSlice(project, j.base.Issues) {
			// fmt.Printf("    skipping time entry (not selected)\n")
			slog.Debug("skipping issue since it is not selected", "issue", project)
			continue
		}

		changed, err := j.updateJiraTracking(project, e)
		if err != nil {
			slog.Error("unable to sync issue", "issue", project, "err", err)
			// fmt.Fprintf(os.Stderr, "unable to sync with issue %s: %v", project, err)
			continue
		}
		// Keep track of first change date
		if changed && firstChange.After(*e.Start) {
			firstChange = *e.Start
		}
		// Keep track of how many entries already exist
		if !changed {
			alreadyExistEntries++
		} else {
			c <- drivers.SyncedEntry{
				Date:     e.Start.Format("2006/01/02 Mon"),
				Duration: e.Duration,
				Type:     drivers.Simple,
				Message:  fmt.Sprintf("[%s] inserted %d from Toggl entry %d to %s's worklog entry", e.Start.Format("15:04"), e.Duration, e.ID, project),
			}
		}
		// fmt.Println()

		// if j.base.DryRun && alreadyExistEntries > 0 {
		// 	fmt.Printf("You can insert the above unsynced events faster with: %s\n", getSuggest(fromDate, firstChange))
		// }
	}
	return nil
}

// func getSuggest(from string, firstChange time.Time) string {
// 	suggest := []string{}
// 	toSeen := false
// 	for _, a := range os.Args {
// 		if a == "-to" || a == "--to" {
// 			toSeen = true
// 		}
// 		if a == "-n" {
// 			continue
// 		}
// 		if a == from {
// 			suggest = append(suggest, firstChange.Format("200601021504"))
// 			continue
// 		}
// 		suggest = append(suggest, a)
// 	}

// 	// If no "-to" was present in the original command
// 	// explicitely set the end date to today
// 	if !toSeen {
// 		suggest = append(suggest, "--to", "today")
// 	}

// 	return strings.Join(suggest, " ")
// }

func (j Jira) updateJiraTracking(issueID string, togglEntry toggl.TimeEntry) (bool, error) {
	tp := jira.BasicAuthTransport{
		Username: config.Current.Get("jira.username"),
		Password: config.Current.Get("jira.token"),
	}
	jiraClient, _ := jira.NewClient(tp.Client(), config.Current.Get("jira.url"))
	wl, _, err := jiraClient.Issue.GetWorklogs(issueID)

	if err != nil {
		return false, err
	}

	// Search worklog for existing entries so we're idempotent
	// Entries contain with `toggl_id: ID` to link to toggl entries
	for _, wlr := range wl.Worklogs {
		search := fmt.Sprintf("toggl_id: %d", togglEntry.ID)
		re := regexp.MustCompile(search)
		matches := re.FindStringSubmatch(wlr.Comment)
		if len(matches) > 0 {
			slog.Info("worklog entry already exists", "entry", issueID, "toggle.ID", togglEntry.ID, "time", wlr.TimeSpent)
			return false, nil
		}
	}

	// Prepare human readable time representation
	dur := time.Duration(time.Duration(togglEntry.Duration) * time.Second)
	// Round entry to the minute above
	// We do not use Truncate since it does not work for Local times
	if time.Duration(togglEntry.Duration)%60 != 0 {
		dur += (60 - time.Duration(togglEntry.Duration)%60) * time.Second
	}

	// and also a Jira-readable one
	durText := fmt.Sprintf("%dh %dm", int(dur.Hours()), int(dur.Minutes())%60)

	// Human readable duration requires checking days difference, etc...
	refStart := togglEntry.StartTime().Local()
	refStop := togglEntry.StopTime().Local()

	// check if j.base.TimeZone is set to UTC
	if j.base.TimeZone != time.UTC {
		refStart = togglEntry.StartTime().UTC()
		refStop = togglEntry.StopTime().UTC()
	}

	startText := refStart.Format("15:04")
	stopText := refStop.Format("15:04")

	// Get difference in days between start and stop
	days := refStop.Sub(refStart).Hours() / 24
	// Add 1 day if task has been stopped after midnight
	if refStop.Hour() < refStart.Hour() {
		days++
	}
	if days >= 1 {
		stopText = fmt.Sprintf("%s j+%d", stopText, int(days))
	}

	comment := strings.ReplaceAll(togglEntry.Description, issueID, "")
	comment = strings.Trim(comment, " ")
	commentInIssue := false

	if j.base.DryRun {
		slog.Info("would insert", "start", startText,
			"stop", stopText, "duration", durText,
			"toggl.ID", togglEntry.ID, "issue", issueID)

		// if interactive {
		// 	fmt.Println("                    asking message interactively")
		// } else {
		// 	fmt.Printf("                    using auto message: %s\n", comment)
		// }

		return true, nil
	}

	// if interactive {
	// 	reader := bufio.NewReader(os.Stdin)
	// 	prompt := fmt.Sprintf("    [%s - %s] (%s) %s comment -",
	// 		startText,
	// 		stopText,
	// 		durText,
	// 		issueID,
	// 	)
	// 	for {
	// 		fmt.Printf("%s> ", prompt)
	// 		line, _ := reader.ReadString('\n')
	// 		if line == "\n" {
	// 			break
	// 		}
	// 		comment += line
	// 		// prompt for next lines is made of spaces
	// 		prompt = strings.Repeat(" ", len(prompt))
	// 	}
	// }

	if len(comment) > 0 && comment[0] == '*' {
		commentInIssue = true
		comment = strings.TrimSpace(comment[1:])
	}

	jTime := jira.Time(*togglEntry.Start)
	jsTime := jira.Time(*togglEntry.Start)
	jComment := fmt.Sprintf("toggl_id: %d\n%s", togglEntry.ID, comment)

	// Ensure we have at leat 60 seconds or Jira will complain
	if togglEntry.Duration < 60 {
		togglEntry.Duration = 60
	}
	wlr := &jira.WorklogRecord{
		TimeSpentSeconds: int(togglEntry.Duration),
		Created:          &jTime,
		Started:          &jsTime,
		Comment:          jComment,
	}

	_, _, err = jiraClient.Issue.AddWorklogRecord(issueID, wlr)
	if err != nil {
		slog.Error("unable to insert worklog entry in issue", "issue", issueID, "test", durText, "toggl.ID", togglEntry.ID, "err", err)
		return false, err
	}

	slog.Info("inserting worklog entry in issue", "issue", issueID, "test", durText, "toggl.ID", togglEntry.ID)
	// fmt.Printf("    [%s - %s] inserted %s from Toggl entry %d to %s's worklog entry\n",
	// 	startText,
	// 	stopText,
	// 	durText,
	// 	togglEntry.ID,
	// 	issueID,
	// )

	if commentInIssue {
		issueComment := &jira.Comment{
			Body: comment,
		}
		_, _, err = jiraClient.Issue.AddComment(issueID, issueComment)

		if err != nil {
			slog.Error("unable to insert comment in issue", "issue", issueID, "err", err)
			return false, err
		}

		// type Comment struct {
		// 	ID           string            `json:"id,omitempty" structs:"id,omitempty"`
		// 	Self         string            `json:"self,omitempty" structs:"self,omitempty"`
		// 	Name         string            `json:"name,omitempty" structs:"name,omitempty"`
		// 	Author       User              `json:"author,omitempty" structs:"author,omitempty"`
		// 	Body         string            `json:"body,omitempty" structs:"body,omitempty"`
		// 	UpdateAuthor User              `json:"updateAuthor,omitempty" structs:"updateAuthor,omitempty"`
		// 	Updated      string            `json:"updated,omitempty" structs:"updated,omitempty"`
		// 	Created      string            `json:"created,omitempty" structs:"created,omitempty"`
		// 	Visibility   CommentVisibility `json:"visibility,omitempty" structs:"visibility,omitempty"`
		// }

	}

	return true, nil
}

// changed, err := updateJiraRollup(day, issue, dur)
func (j *Jira) updateJiraRollup(day, issueID, description string, seconds int64) (bool, error) {
	tp := jira.BasicAuthTransport{
		Username: config.Current.Get("jira.username"),
		Password: config.Current.Get("jira.token"),
	}
	jiraClient, _ := jira.NewClient(tp.Client(), config.Current.Get("jira.url"))
	wl, _, err := jiraClient.Issue.GetWorklogs(issueID)

	if err != nil {
		return false, err
	}

	dur := time.Duration(seconds * int64(time.Second))
	// Search worklog for existing entries so we're idempotent
	// Entries contain with `toggl_id: ID` to link to toggl entries
	ref := strings.ReplaceAll(day, "/", "-")
	for _, wlr := range wl.Worklogs {
		search := fmt.Sprintf("rollup: %s/%s", tp.Username, ref)
		re := regexp.MustCompile(search)
		matches := re.FindStringSubmatch(wlr.Comment)
		if len(matches) > 0 {
			slog.Debug("worklog rollup entry already exists", "issue", issueID, "ref", ref)
			return false, nil
		}
	}

	// Prepare Jira-readable time duration
	durText := fmt.Sprintf("%dh %dm", int(dur.Hours()), int(dur.Minutes())%60)

	if j.base.DryRun {
		slog.Info("dry run mode entry", "day", day, "issue", issueID, "description", description, "duration", durText)

		// if interactive {
		// 	fmt.Println("                    asking confirmation interactively")
		// }

		return true, nil
	}

	// if interactive {
	// 	reader := bufio.NewReader(os.Stdin)
	// 	fmt.Printf("    insert woklog entry for %s %q (%s) [y/n] ? ",
	// 		issueID,
	// 		description,
	// 		durText,
	// 	)
	// 	line, _ := reader.ReadString('\n')
	// 	line = strings.TrimSpace(line)
	// 	if line != "y" {
	// 		return true, nil
	// 	}
	// }

	startTime, err := time.Parse("2006/01/02 Mon", day)
	if err != nil {
		return false, err
	}
	jTime := jira.Time(startTime)
	jsTime := jira.Time(startTime)
	jComment := fmt.Sprintf("%s (rollup: %s/%s)", description, tp.Username, ref)

	wlr := &jira.WorklogRecord{
		TimeSpentSeconds: int(dur / time.Second),
		Created:          &jTime,
		Started:          &jsTime,
		Comment:          jComment,
	}

	_, _, err = jiraClient.Issue.AddWorklogRecord(issueID, wlr)
	if err != nil {
		slog.Error("unable to insert rollup entry in issue worklog", "issue", issueID, "test", durText, "rollup", ref, "err", err)
		// fmt.Printf("    unable to insert %s from rollup entry %s to %s's worklog entry: %v", durText, ref, issueID, err)
		return false, err
	}
	slog.Info("inserted rollup entry in issue worklog", "issue", issueID, "test", durText, "rollup", ref)

	// fmt.Printf("    [%s] inserted %s from rollup entry to %s's worklog entry\n",
	// 	ref,
	// 	durText,
	// 	issueID,
	// )

	return true, nil
}

func isInSlice(entry string, sl []string) bool {
	for _, e := range sl {
		if e == entry {
			return true
		}
	}

	return false
}

func extractTicketFromString(e string) string {
	exp := `[A-Z]+-\d+`

	re := regexp.MustCompile(exp)

	project := string(re.Find([]byte(e)))
	project = strings.TrimSpace(project)

	return string(project)
}
