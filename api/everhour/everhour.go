package everhour

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
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
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Projects   []string   `json:"projects"`
	Section    int        `json:"section"`
	Labels     []string   `json:"labels"`
	Position   int        `json:"position"`
	DueAt      string     `json:"dueAt"`
	Status     string     `json:"status"`
	Time       Time       `json:"time"`
	Estimate   Estimate   `json:"estimate"`
	Attributes Attributes `json:"attributes"`
	Metrics    Metrics    `json:"metrics"`
}

type TimeRecord struct {
	ID         int       `json:"id"`
	Time       int       `json:"time"`
	User       int       `json:"user"`
	Date       string    `json:"date"`
	Task       Task      `json:"task"`
	IsLocked   bool      `json:"isLocked"`
	IsInvoiced bool      `json:"isInvoiced"`
	Comment    string    `json:"comment"`
	History    []History `json:"history"`
}

type Time struct {
	Total int            `json:"total"`
	Users map[string]int `json:"users"`
}

type Estimate struct {
	Total int            `json:"total"`
	Type  string         `json:"type"`
	Users map[string]int `json:"users"`
}

type Attributes map[string]string

type Metrics struct {
	Efforts  int `json:"efforts"`
	Expenses int `json:"expenses"`
}

type History struct {
	ID           int    `json:"id"`
	CreatedBy    int    `json:"createdBy"`
	Time         int    `json:"time"`
	PreviousTime int    `json:"previousTime"`
	Action       string `json:"action"`
	CreatedAt    string `json:"createdAt"`
}

type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	HeadLine  string `json:"headline"`
	AvatarURL string `json:"avatarUrl"`
	Role      string `json:"role"`
	Status    string `json:"status"`
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

// GetTask returns a task by ID
func (c *Client) GetTask(id string) (Task, error) {
	var task Task

	req, err := c.buildRequest("GET", "/tasks/"+id, nil)
	if err != nil {
		return task, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return task, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&task)
	if err != nil {
		return task, err
	}

	return task, nil
}

// GetTaskByName returns a task by name
func (c *Client) GetTaskByName(name, project string) (*Task, error) {
	var tasks []Task

	s := fmt.Sprintf("/tasks/search?query=%s&limit=10&searchInClosed=false", url.QueryEscape(name))
	req, err := c.buildRequest("GET", s, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&tasks)
	if err != nil {
		return nil, err
	}

	// Loop through tasks to find the one with the right project
	for _, t := range tasks {
		if slices.Contains(t.Projects, project) {
			return &t, nil
		}
	}

	return nil, nil
}

// GetTaskTimeRecords returns time records for task
func (c *Client) GetTaskTimeRecords(id string) ([]TimeRecord, error) {
	var tr []TimeRecord

	req, err := c.buildRequest("GET", "/tasks/"+id+"/time", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&tr)
	if err != nil {
		return nil, err
	}

	return tr, nil
}

// AddTime creates a time record for a task
func (c *Client) AddTime(taskID string, date *time.Time, time int64, comment string) error {
	type addTimeRequest struct {
		Time    int64  `json:"time"`
		Date    string `json:"date"`
		User    int    `json:"user"`
		Comment string `json:"comment"`
	}

	u, err := c.GetCurrentUser()
	if err != nil {
		return err
	}
	if u == nil {
		return fmt.Errorf("could not get current user")
	}

	entry := addTimeRequest{
		Time:    time,
		Date:    date.Format("2006-01-02"),
		User:    u.ID,
		Comment: strings.TrimSpace(comment),
	}

	reqBody, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	req, err := c.buildRequest("POST", "/tasks/"+taskID+"/time", reqBody)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// dump body if we got an error
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error adding time: %s", body)
	}

	return nil
}

// GetCurrentUser returns the current user
func (c *Client) GetCurrentUser() (*User, error) {
	var user User

	req, err := c.buildRequest("GET", "/users/me", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *Client) buildRequest(method, url string, body []byte) (*http.Request, error) {
	// create io.Reader from bytes
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, API_URL+url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.token)
	return req, nil
}
