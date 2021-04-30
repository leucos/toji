package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	jira "github.com/andygrunwald/go-jira"
	ct "github.com/daviddengcn/go-colortext"
	toggl "github.com/jason0x43/go-toggl"
)

var validArgs = []string{
	"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday",
	"mon", "tue", "wed", "thu", "fri", "sat", "sun",
	"today", "yesterday",
	"week", "month", "year",
}

var syncCmd = &cobra.Command{
	Use:     "sync <start>",
	Short:   "syncs time entries from toggl to jira",
	Example: "toji sync yesterday --to today",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		return doSync(args[0])
	},
	// SilenceUsage: true,
	ValidArgs: validArgs,
}

var (
	toDate      string
	dryRun      bool
	utc         bool
	interactive bool
	onlyIssues  []string
)

func init() {
	syncCmd.Flags().StringVarP(&toDate, "to", "t", "", "ending date")
	syncCmd.Flags().BoolVarP(&dryRun, "dryrun", "n", false, "do not update Jira entries")
	syncCmd.Flags().BoolVarP(&utc, "utc", "u", false, "display entries using UTC in the terminal")
	syncCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "asks a comment for each worklog interactively")
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

	projectList := []string{}
	// if we have filters, prepare a string slice
	if getConfig("jira.projects") != "" {
		projectList = strings.Split(getConfig("jira.projects"), ",")
	}

	fmt.Printf("\nSyncing toggl entries between %s and %s\n", from, to)

	session := toggl.OpenSession(getConfig("toggle.token"))
	entries, err := session.GetTimeEntries(from, to)

	if err != nil {
		return fmt.Errorf("unable to fetch Toggl entries: %v. Is your token valid ?", err)
	}

	currentDate := from.AddDate(-1, 0, 0).Format("Mon 2006/01/02")
	currentProject := ""

	firstChange := time.Now()
	alreadyExistEntries := 0

	for _, e := range entries {
		textDate := e.Start.Format("Mon 2006/01/02")
		if textDate != currentDate {
			fmt.Printf("\n%s\n==============\n", textDate)
			currentDate = textDate
			currentProject = ""
		}

		// Project holds the Jira ticket ID (e.g. XYZ-123)
		project := getTicketFromEntry(e.Description)

		fmt.Printf("")
		if project == "" {
			continue
		}

		// if we have project filters, check if we have a match
		if len(projectList) > 0 {
			projectSlug := strings.Split(project, "-")
			if !isInSlice(projectSlug[0], projectList) {
				ct.Foreground(ct.Cyan, true)
				fmt.Printf("    skipping since project not included for entry %s\n", project)
				ct.ResetColor()
				continue
			}
		}

		// Only redisplay project description if the project is not the same as
		// previous iteration
		if project != currentProject {
			fmt.Printf("\n  %s (%s/browse/%s)\n", e.Description, getConfig("jira.url"), project)
			currentProject = project
		}

		if e.StopTime().IsZero() {
			ct.Foreground(ct.Cyan, true)
			fmt.Printf("    skipping currently running time entry for %s\n", project)
			ct.ResetColor()
			continue
		}

		if onlyIssues != nil && !isInSlice(project, onlyIssues) {
			ct.Foreground(ct.Red, false)
			fmt.Printf("    skipping time entry (not selected)\n")
			ct.ResetColor()
			continue
		}

		changed, err := updateJiraTracking(project, e)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to sync with issue %s: %v", project, err)
			continue
		}
		// Keep track of fist change date
		if changed && firstChange.After(*e.Start) {
			firstChange = *e.Start
		}
		// Keep track of how many entries already exist
		if !changed {
			alreadyExistEntries++
		}
	}
	fmt.Println()

	if dryRun && alreadyExistEntries > 0 {
		fmt.Printf("You can insert the above unsynced events faster with: %s\n", getSuggest(fromDate, firstChange))
	}
	return nil
}

func getSuggest(from string, firstChange time.Time) string {
	suggest := []string{}
	toSeen := false
	for _, a := range os.Args {
		if a == "-to" || a == "--to" {
			toSeen = true
		}
		if a == "-n" {
			continue
		}
		if a == from {
			suggest = append(suggest, firstChange.Format("200601021504"))
			continue
		}
		suggest = append(suggest, a)
	}

	// If no "-to" was present in the original command
	// explicitely set the end date to today
	if !toSeen {
		suggest = append(suggest, "--to", "today")
	}

	return strings.Join(suggest, " ")
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
		// If only YYYYmmDD is specified start at 00h00
		// time.Truncate can't be used since it works only for UTC
		if d, err := time.Parse("20060102", s); err == nil {
			start = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
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
		// If only YYYYmmDD is specified start at 00h00
		// time.Truncate can't be used since it works only for UTC
		if d, err := time.Parse("20060102", e); err == nil {
			end = time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 0, 0, time.Local)
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
	exp := `[A-Z]+-\d+`

	re := regexp.MustCompile(exp)

	project := string(re.Find([]byte(e)))
	project = strings.TrimSpace(project)

	return string(project)
}

func updateJiraTracking(issueID string, togglEntry toggl.TimeEntry) (bool, error) {
	tp := jira.BasicAuthTransport{
		Username: getConfig("jira.username"),
		Password: getConfig("jira.token"),
	}
	jiraClient, _ := jira.NewClient(tp.Client(), getConfig("jira.url"))
	wl, _, err := jiraClient.Issue.GetWorklogs(issueID)

	if err != nil {
		return false, err
	}

	// Search worklog for existing entries so we're idempotent
	// Entries contain with `toggl_id: ID` to link to toggle entries
	for _, wlr := range wl.Worklogs {
		search := fmt.Sprintf("toggl_id: %d", togglEntry.ID)
		re := regexp.MustCompile(search)
		matches := re.FindStringSubmatch(wlr.Comment)
		if len(matches) > 0 {
			ct.Foreground(ct.Blue, false)
			fmt.Printf("    worklog entry %s for Toggle entry %d (%s) already exists\n", issueID, togglEntry.ID, wlr.TimeSpent)
			ct.ResetColor()
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

	if utc {
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

	if dryRun {
		ct.Foreground(ct.Yellow, false)
		fmt.Printf("    [%s - %s] would insert %s from Toggl entry %d to %s's worklog entry\n",
			startText,
			stopText,
			durText,
			togglEntry.ID,
			issueID,
		)
		if interactive {
			fmt.Println("                    asking message interactively")
		} else {
			fmt.Printf("                    using auto message: %s\n", comment)
		}
		ct.ResetColor()

		return true, nil
	}

	if interactive {
		reader := bufio.NewReader(os.Stdin)
		prompt := fmt.Sprintf("    [%s - %s] (%s) %s comment -",
			startText,
			stopText,
			durText,
			issueID,
		)
		for {
			fmt.Printf("%s> ", prompt)
			line, _ := reader.ReadString('\n')
			if line == "\n" {
				break
			}
			comment += line
			// prompt for next lines is made of spaces
			prompt = strings.Repeat(" ", len(prompt))
		}
	}

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
		fmt.Printf("    unable to insert %s from Toggl entry %d to %s's worklog entry: %v", durText, togglEntry.ID, issueID, err)
		return false, err
	}

	ct.Foreground(ct.Yellow, false)
	fmt.Printf("    [%s - %s] inserted %s from Toggl entry %d to %s's worklog entry\n",
		startText,
		stopText,
		durText,
		togglEntry.ID,
		issueID,
	)
	ct.ResetColor()

	if commentInIssue {
		issueComment := &jira.Comment{
			Body: comment,
		}
		_, _, err = jiraClient.Issue.AddComment(issueID, issueComment)

		if err != nil {
			fmt.Printf("    unable to insert comment in issue %s: %v", issueID, err)
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

func isInSlice(entry string, sl []string) bool {
	for _, e := range sl {
		if e == entry {
			return true
		}
	}

	return false
}
