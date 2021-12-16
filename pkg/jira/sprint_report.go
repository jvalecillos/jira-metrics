package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path"
)

type Fieldvalue struct {
	Value float64 `json:"value,omitempty"`
	Text  string  `json:"text,omitempty"`
}

type Statistic struct {
	StatFieldID    string     `json:"statFieldId"`
	StatFieldValue Fieldvalue `json:"statFieldValue,omitempty"`
}

type Epic struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	Editable      bool   `json:"editable"`
	Renderer      string `json:"renderer"`
	IssueID       int    `json:"issueId"`
	EpicKey       string `json:"epicKey"`
	EpicColor     string `json:"epicColor"`
	Text          string `json:"text"`
	CanRemoveEpic bool   `json:"canRemoveEpic"`
}

type StatusCategory struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	ColorName string `json:"colorName"`
}

type Status struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	IconURL        string         `json:"iconUrl"`
	StatusCategory StatusCategory `json:"statusCategory"`
}

type Issue struct {
	ID                        int       `json:"id"`
	Key                       string    `json:"key"`
	Hidden                    bool      `json:"hidden"`
	TypeName                  string    `json:"typeName"`
	TypeID                    string    `json:"typeId"`
	Summary                   string    `json:"summary"`
	TypeURL                   string    `json:"typeUrl"`
	PriorityURL               string    `json:"priorityUrl"`
	PriorityName              string    `json:"priorityName"`
	Done                      bool      `json:"done"`
	Assignee                  string    `json:"assignee,omitempty"`
	AssigneeKey               string    `json:"assigneeKey,omitempty"`
	AssigneeAccountID         string    `json:"assigneeAccountId,omitempty"`
	AssigneeName              string    `json:"assigneeName,omitempty"`
	AvatarURL                 string    `json:"avatarUrl,omitempty"`
	HasCustomUserAvatar       bool      `json:"hasCustomUserAvatar"`
	Flagged                   bool      `json:"flagged"`
	Epic                      string    `json:"epic,omitempty"`
	EpicField                 Epic      `json:"epicField,omitempty"`
	ColumnStatistic           Statistic `json:"columnStatistic"`
	CurrentEstimateStatistic  Statistic `json:"currentEstimateStatistic"`
	EstimateStatisticRequired bool      `json:"estimateStatisticRequired"`
	EstimateStatistic         Statistic `json:"estimateStatistic"`
	StatusID                  string    `json:"statusId"`
	StatusName                string    `json:"statusName"`
	StatusURL                 string    `json:"statusUrl"`
	Status                    Status    `json:"status"`
	FixVersions               []int     `json:"fixVersions"`
	ProjectID                 int       `json:"projectId"`
	LinkedPagesCount          int       `json:"linkedPagesCount"`
}

type EstimateSum Fieldvalue

type IssuesAdded map[string]bool

type Contents struct {
	CompletedIssues                   []Issue `json:"completedIssues"`
	IssuesNotCompletedInCurrentSprint []Issue `json:"issuesNotCompletedInCurrentSprint"`
	// Removed from Sprint
	PuntedIssues []Issue `json:"puntedIssues"`
	// Completed outside the Sprint
	IssuesCompletedInAnotherSprint                   []Issue     `json:"issuesCompletedInAnotherSprint"`
	CompletedIssuesInitialEstimateSum                EstimateSum `json:"completedIssuesInitialEstimateSum"`
	CompletedIssuesEstimateSum                       EstimateSum `json:"completedIssuesEstimateSum"`
	IssuesNotCompletedInitialEstimateSum             EstimateSum `json:"issuesNotCompletedInitialEstimateSum"`
	IssuesNotCompletedEstimateSum                    EstimateSum `json:"issuesNotCompletedEstimateSum"`
	AllIssuesEstimateSum                             EstimateSum `json:"allIssuesEstimateSum"`
	PuntedIssuesInitialEstimateSum                   EstimateSum `json:"puntedIssuesInitialEstimateSum"`
	PuntedIssuesEstimateSum                          EstimateSum `json:"puntedIssuesEstimateSum"`
	IssuesCompletedInAnotherSprintInitialEstimateSum EstimateSum `json:"issuesCompletedInAnotherSprintInitialEstimateSum"`
	IssuesCompletedInAnotherSprintEstimateSum        EstimateSum `json:"issuesCompletedInAnotherSprintEstimateSum"`
	IssueKeysAddedDuringSprint                       IssuesAdded `json:"issueKeysAddedDuringSprint"`
}

type Sprint struct {
	ID               int    `json:"id"`
	Sequence         int    `json:"sequence"`
	Name             string `json:"name"`
	State            string `json:"state"`
	LinkedPagesCount int    `json:"linkedPagesCount"`
	Goal             string `json:"goal"`
	StartDate        string `json:"startDate"`
	EndDate          string `json:"endDate"`
	IsoStartDate     string `json:"isoStartDate"`
	IsoEndDate       string `json:"isoEndDate"`
	CompleteDate     string `json:"completeDate"`
	IsoCompleteDate  string `json:"isoCompleteDate"`
	CanUpdateSprint  bool   `json:"canUpdateSprint"`
	// TODO: it is not clear purpose of this field, currently an empty array in the sample
	// RemoteLinks      []interface{} `json:"remoteLinks"`
	DaysRemaining int `json:"daysRemaining"`
}

type ReportResponse struct {
	Contents        Contents `json:"contents"`
	Sprint          Sprint   `json:"sprint"`
	LastUserToClose string   `json:"lastUserToClose"`
	SupportsPages   bool     `json:"supportsPages"`
}

const (
	// SprintReporSuffix used for getting sprint report
	SprintReporSuffix = "/rest/greenhopper/1.0/rapid/charts/sprintreport"
)

// SprintReport contains the logic to use JIRA Sprint Report hidden API
type SprintReport struct {
	*Jira
}

// Report wraps reporting API
func (a *Jira) Report() (*SprintReport, error) {
	return &SprintReport{a}, nil
}

// sprintReportURL returns the base URL for the JIRA Sprint Report hidden API
func (a *SprintReport) sprintReportURL() (*url.URL, error) {
	u, err := url.Parse(a.EndpointPrefix)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, SprintReporSuffix)
	return u, nil

}

// Get fetches sprint report from Jira API
func (a *SprintReport) Get(ctx context.Context, rapidViewId string, sprintId string) (*ReportResponse, error) {

	url, err := a.sprintReportURL()
	if err != nil {
		return nil, err
	}

	// Adding GET parameters
	q := url.Query()
	q.Add("rapidViewId", rapidViewId)
	q.Add("sprintId", sprintId)
	// Encode and assign back to the original query.
	url.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)

	if err != nil {
		return nil, err
	}

	resp, err := a.execute(ctx, req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var ar ReportResponse
	err = json.NewDecoder(resp.Body).Decode(&ar)
	if err != nil {
		return nil, err
	}

	return &ar, nil
}
