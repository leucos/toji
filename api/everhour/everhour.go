package everhour

import (
	"encoding/json"
	"io"
	"net/http"
)

const API_URL = "https://api.everhour.com"

type Client struct {
	token string
}

type Project struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	WorkSpaceID   string `json:"workspaceId"`
	WorkSpaceName string `json:"workspaceName"`
	ClientID      int    `json:"client"`
	Type          string `json:"type"`
	Users         []int  `json:"users"`
}

type Task struct {
	ID          string
	Name        string
	Description string
	ProjectIDs  []string
	SectionID   string
	Time        int // time for current user only
}

type User struct {
	ID   int
	Name string
}

func New(token string) *Client {
	return &Client{token: token}
}

func (c *Client) GetAllProjects() ([]Project, error) {
	var (
		projects []Project
	)

	req, err := c.buildRequest("GET", "/projects?limit=100", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&projects)
	if err != nil {
		return nil, err
	}

	return projects, nil
}

func (c *Client) GetCurrentUser() (User, error) {
	return User{}, nil
}

func (c *Client) buildRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, API_URL+url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.token)
	return req, nil
}
