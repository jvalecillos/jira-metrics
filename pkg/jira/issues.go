package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path"
)

// Component associated with the task
type Component struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Discipline custom field for type of engineering task
type Discipline struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// IssueType for JIRA tickets
type IssueType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Subtask     bool   `json:"subtask"`
}

// Fields inside a JIRA issue
type Fields struct {
	Components  []Component `json:"components"`
	StoryPoints float64     `json:"customfield_10005"`
	ParentKey   string      `json:"customfield_10009"`
	Discipline  Discipline  `json:"customfield_12142"`
	Issuetype   IssueType   `json:"issuetype"`
	Summary     string      `json:"summary"`
}

// SimpleIssue simplified version of API response for issues
type SimpleIssue struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields Fields `json:"fields"`
}

const (
	// IssueDetailsSuffix used for getting issue details
	IssueDetailsSuffix = "/rest/api/latest/issue/"
)

// IssueDetails contains the logic to use JIRA issues API endpoints
type IssueDetails struct {
	*Jira
}

// Report wraps Jira Issues API
func (a *Jira) Issues() (*IssueDetails, error) {
	return &IssueDetails{a}, nil
}

// sprintReportURL returns the base URL for the JIRA Sprint Report hidden API
func (a *IssueDetails) issueDetailsURL(issueId string) (*url.URL, error) {
	u, err := url.Parse(a.EndpointPrefix)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, IssueDetailsSuffix, issueId)
	return u, nil

}

// Get fetches issue details from Jira API
func (a *IssueDetails) Get(ctx context.Context, issueId string) (*SimpleIssue, error) {

	url, err := a.issueDetailsURL(issueId)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)

	if err != nil {
		return nil, err
	}

	resp, err := a.execute(ctx, req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var ar SimpleIssue
	err = json.NewDecoder(resp.Body).Decode(&ar)
	if err != nil {
		return nil, err
	}

	return &ar, nil
}
