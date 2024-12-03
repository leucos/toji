package toggl

import (
	"fmt"

	"github.com/jason0x43/go-toggl"
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

func GetProjectNameFromID(session toggl.Session, workspace int, id int) (string, error) {
	// fmt.Printf("fetching projects from workspace: %d\n", workspace)
	project, err := session.GetProject(id, workspace)
	if err != nil {
		return "", fmt.Errorf("unable to fetch Toggl projects for workspace %d: %v. Is your token valid ?", workspace, err)
	}

	if (project != toggl.Project{}) {
		return project.Name, nil
	}

	return "", fmt.Errorf("unable to find project with ID %d", id)
}
