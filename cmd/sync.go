package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	jira "github.com/andygrunwald/go-jira"
	toggl "github.com/jason0x43/go-toggl"
)

var validArgs = []string{
	"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday",
	"mon", "tue", "wed", "thu", "fri", "sat", "sun",
	"today", "yesterday",
	"week", "month", "year",
}

var syncCmd = &cobra.Command{
	Use:     "sync",
	Short:   "syncs time entries from toggl to jira",
	Example: "toji sync yesterday --to today",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		return doSync(args[0])
	},
	SilenceUsage: true,
	ValidArgs:    validArgs,
}

var (
	toDate     string
	dryRun     bool
	onlyIssues []string
)

func init() {
	syncCmd.Flags().StringVarP(&toDate, "to", "t", "", "ending date")
	syncCmd.Flags().BoolVarP(&dryRun, "dryrun", "n", false, "do not update Jira entries")
	syncCmd.Flags().StringSliceVarP(&onlyIssues, "only", "o", nil, "only update these comma-separated entries")

	syncCmd.RegisterFlagCompletionFunc("to", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validArgs, cobra.ShellCompDirectiveDefault
	})

	toggl.DisableLog()
	checkProfile()
}

func doSync(fromDate string) error {
	if toDate == "" {
		toDate = fromDate
	}

	from, to, err := parseTimeSpec(fromDate, toDate)
	if err != nil {
		return fmt.Errorf("unable to parse time using provided '%s' or '%s': %v", fromDate, toDate, err)
	}

	fmt.Printf("\nSyncing toggl entries between %s and %s\n", from, to)

	session := toggl.OpenSession(getConfig("toggle.token"))
	entries, err := session.GetTimeEntries(from, to)

	if err != nil {
		return fmt.Errorf("unable to fetch Toggl entries: %v. Is your token valid ?", err)
	}

	currentDate := from.AddDate(-1, 0, 0).Format("Mon 2006/01/02")
	currentProject := ""

	for _, e := range entries {
		textDate := e.Start.Format("Mon 2006/01/02")
		if textDate != currentDate {
			fmt.Printf("\n%s\n==============\n", textDate)
			currentDate = textDate
			currentProject = ""
		}

		project := getTicketFromEntry(e.Description)

		if project == "" {
			continue
		}

		if onlyIssues != nil && !isInSlice(project, onlyIssues) {
			fmt.Printf("\t%s\n\t\tskipping time entry (not selected)\n", project)
			continue
		}

		// Only redisplay project description if the project is not the same as
		// previous iteration
		if project != currentProject {
			fmt.Printf("\n\t%s\n", e.Description)
			currentProject = project
		}

		err := updateJiraTracking(project, e)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to sync with issue %s: %v ", project, err)
			continue
		}
	}
	return nil
}

func parseTimeSpec(s string, e string) (time.Time, time.Time, error) {
	start := time.Now()
	end := time.Now()

	week := map[string]time.Weekday{
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
		"sunday":    time.Sunday,
		"mon":       time.Monday,
		"tue":       time.Tuesday,
		"wed":       time.Wednesday,
		"thu":       time.Thursday,
		"fri":       time.Friday,
		"sat":       time.Saturday,
		"sun":       time.Sunday,
	}

	switch s {
	case "today":
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	case "yesterday":
		start = start.AddDate(0, 0, -1)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	case "week", "monday":
		for start.Weekday() != time.Monday { // iterate back to Monday
			start = start.AddDate(0, 0, -1)
		}
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	case "month":
		start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.Local)
	case "year":
		start = time.Date(start.Year(), 1, 1, 0, 0, 0, 0, time.Local)
	default: // we got a weekday or a date
		if d, ok := week[s]; ok {
			for start.Weekday() != d { // iterate back to requested day
				start = start.AddDate(0, 0, -1)
			}
			start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
			break
		}
		if d, err := time.Parse("200601021504", s); err == nil {
			start = d
			break
		}
		return start, end, fmt.Errorf("unable to parse start date (%s)", end)
	}

	switch e {
	case "today":
		end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, time.Local)
	case "yesterday":
		end = end.AddDate(0, 0, -1)
		end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, time.Local)
	case "week":
		for end.Weekday() != time.Monday { // iterate back to Monday
			end = end.AddDate(0, 0, -1)
		}
		end = time.Date(end.Year(), end.Month(), end.Day()+6, 23, 59, 59, 0, time.Local)
	case "month":
		end = time.Date(end.Year(), end.Month()+1, 1, 23, 59, 59, 0, time.Local)
		end = end.AddDate(0, 0, -1)
	case "year":
		end = time.Date(end.Year()+1, 1, 1, 23, 59, 59, 0, time.Local)
		end = end.AddDate(0, 0, -1)
	default: // we got a weekday or a date
		if d, ok := week[e]; ok {
			for end.Weekday() != d { // iterate back to requested day
				end = end.AddDate(0, 0, -1)
			}
			end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, time.Local)
			break
		}
		if d, err := time.Parse("200601021504", e); err == nil {
			end = d
			break
		}
		return start, end, fmt.Errorf("unable to parse end date (%s)", end)
	}

	if start.After(end) {
		return start, end, fmt.Errorf("end date (%s) is before start date (%s)", end, start)
	}

	return start, end, nil
}

func getTicketFromEntry(e string) string {
	exp := `[A-Z]+-\d+\s`

	re := regexp.MustCompile(exp)

	project := string(re.Find([]byte(e)))
	project = strings.TrimSpace(project)

	return string(project)
}

func updateJiraTracking(issueID string, togglEntry toggl.TimeEntry) error {
	tp := jira.BasicAuthTransport{
		Username: getConfig("jira.username"),
		Password: getConfig("jira.token"),
	}
	jiraClient, _ := jira.NewClient(tp.Client(), getConfig("jira.url"))
	wl, _, err := jiraClient.Issue.GetWorklogs(issueID)

	if err != nil {
		return err
	}

	for _, wlr := range wl.Worklogs {
		re := regexp.MustCompile(`toggl_id: \d+`)
		matches := re.FindStringSubmatch(wlr.Comment)
		if len(matches) > 0 {
			fmt.Printf("\t\tworklog entry %s for %s already exists\n", issueID, wlr.TimeSpent)
			return nil
		}
	}

	// Prepare human readable time representation
	dur := time.Duration(time.Duration(togglEntry.Duration) * time.Second)
	durText := fmt.Sprintf("%dh %dm %ds", int(dur.Hours()), int(dur.Minutes())%60, int(dur.Seconds())%60)

	startText := togglEntry.StartTime().Format("15:04")
	stopText := togglEntry.StopTime().Format("15:04")
	// Get difference in days between start and stop
	days := togglEntry.StopTime().Sub(togglEntry.StartTime()).Hours() / 24
	// Add 1 if task has been stopped after midnight
	if togglEntry.StopTime().Hour() < togglEntry.StartTime().Hour() {
		days++
	}
	if days >= 1 {
		stopText = fmt.Sprintf("%s j+%d", stopText, int(days))
	}

	if dryRun {
		fmt.Printf("\t\t[%s - %s] would insert %s from entry %d to %s's worklog entry\n",
			startText,
			stopText,
			durText,
			togglEntry.ID,
			issueID,
		)
		return nil
	}

	jTime := jira.Time(*togglEntry.Start)
	// jUser, _, _ := jiraClient.User.GetSelf()
	jComment := fmt.Sprintf("toggl_id: %d\n", togglEntry.ID)

	wlr := &jira.WorklogRecord{
		// Author:           jUser,
		TimeSpentSeconds: int(togglEntry.Duration),
		Created:          &jTime,
		Comment:          jComment,
	}
	_, _, err = jiraClient.Issue.AddWorklogRecord(issueID, wlr)
	if err != nil {
		fmt.Printf("\t\tunable to insert %s from entry %d to %s's worklog entry: %v", durText, togglEntry.ID, issueID, err)
		return err
	}

	fmt.Printf("\t\t[%s - %s] inserted %s from entry %d to %s's worklog entry\n",
		startText,
		stopText,
		durText,
		togglEntry.ID,
		issueID,
	)

	return nil
}

func isInSlice(entry string, sl []string) bool {
	for _, e := range sl {
		if e == entry {
			return true
		}
	}

	return false
}
