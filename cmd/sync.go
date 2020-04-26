package cmd

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/spf13/cobra"

	jira "github.com/andygrunwald/go-jira"
	toggl "github.com/jason0x43/go-toggl"
)

var syncCmd = &cobra.Command{
	Use:     "sync",
	Short:   "syncs time entries from toggl to jira",
	Example: "toji sync yesterday --to today",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return doSync(args[0])
	},
	SilenceUsage: true,
}

var (
	toDate     string
	dryRun     bool
	onlyIssues []string
)

func init() {
	syncCmd.Flags().StringVarP(&toDate, "to", "t", "", "ending date")
	syncCmd.Flags().BoolVarP(&dryRun, "dryrun", "n", false, "do not update Jira entries")
	syncCmd.Flags().StringSliceVarP(&onlyIssues, "only", "o", nil, "only update this entry")
	// syncCmd.Flags().BoolVar(&interactive, "interactive", false, "interactive confirmation mode")
	// syncCmd.Flags().BoolVar(&message, "message", false, "asks additional message for every entry")
	// edit command ? syncCmd.Flags().BoolVar(&amend, "amend", false, "amend existing entries")
	// syncCmd.Flags().BoolVar(&devices, "devices", false, "devices cgroup")
	// syncCmd.Flags().BoolVar(&freezer, "freezer", false, "freezer cgroup")
	// syncCmd.Flags().BoolVar(&hugetlb, "hugetlb", false, "hugetlb cgroup")
	// syncCmd.Flags().BoolVar(&memory, "memory", false, "memory cgroup")
	// syncCmd.Flags().BoolVar(&net, "net", false, "net cgroup")
	// syncCmd.Flags().BoolVar(&perfevent, "perfevent", false, "perfevent cgroup")
	// syncCmd.Flags().BoolVar(&pids, "pids", false, "pids cgroup")
	toggl.DisableLog()
}

func doSync(fromDate string) error {
	if toDate == "" {
		toDate = fromDate
	}

	from, to, err := parseTimeSpec(fromDate, toDate)
	if err != nil {
		return err
	}

	fmt.Printf("Syncing toggl entries between %s (%s) and %s (%s)\n", fromDate, from, toDate, to)

	session := toggl.OpenSession(getConfig("toggle.token"))
	entries, err := session.GetTimeEntries(from, to)

	if err != nil {
		return err
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
			fmt.Printf("\t%s\n", e.Description)
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
	}

	if start.After(end) {
		return start, end, fmt.Errorf("end date (%s) is before start date (%s)", end, start)
	}

	return start, end, nil
}

func getTicketFromEntry(e string) string {
	re := regexp.MustCompile(`^[A-Z]+-\d+`)
	project := re.Find([]byte(e))

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

	if dryRun {
		fmt.Printf("\t\twould insert %s from entry %d to %s's worklog entry\n", durText, togglEntry.ID, issueID)
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

	fmt.Printf("\t\tinserted %s from entry %d to %s's worklog entry\n", durText, togglEntry.ID, issueID)
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
