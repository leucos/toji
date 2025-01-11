package toggl

import (
	"fmt"
	"log/slog"
	"regexp"

	"github.com/leucos/go-toggl"
)

func GetProjectIDFromName(session toggl.Session, workspace int, name string) (int, error) {
	projects, err := session.GetProjects(workspace)
	// fmt.Printf("fetching projects from workspace: %d\n", workspace)
	if err != nil {
		return -1, fmt.Errorf("unable to fetch Toggl projects for workspace %d: %v. Is your token valid ?", workspace, err)
	}

	for _, project := range projects {
		if project.Name == name {
			return project.ID, nil
		}
	}

	return -1, fmt.Errorf("unable to find project %s", name)
}

func GetProjectNameFromID(session toggl.Session, wid int, id int) (string, error) {
	slog.Debug("getting project name from id", "wid", wid, "id", id)

	project, err := session.GetProject(id, wid)
	if err != nil {
		return "", fmt.Errorf("unable to fetch Toggl projects for workspace %d: %v. Is your token valid ?", wid, err)
	}

	if (project != toggl.Project{}) {
		return project.Name, nil
	}

	return "", fmt.Errorf("unable to find project with ID %d", id)
}

func CheckIssueMatches(oklist []string, sut string) bool {
	// Optimistic direct match test
	for _, ok := range oklist {
		if ok == sut {
			return true
		}
	}
	// Now test using regexps
	for _, ok := range oklist {
		// create a regexp from the string
		re, err := regexp.Compile(ok)
		if err != nil {
			slog.Error("unable to compile regexp", "regexp", ok, "error", err)
			return false
		}
		if re.MatchString(sut) {
			return true
		}
	}

	return false
}
