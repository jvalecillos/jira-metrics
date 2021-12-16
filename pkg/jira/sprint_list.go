package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

type BasicSprint struct {
	ID               int    `json:"id"`
	Sequence         int    `json:"sequence"`
	Name             string `json:"name"`
	State            string `json:"state"`
	LinkedPagesCount int    `json:"linkedPagesCount"`
	Goal             string `json:"goal"`
}

type SprintsResponse struct {
	Sprints     []BasicSprint `json:"sprints"`
	RapidViewID int           `json:"rapidViewId"`
}

const (
	// IssueDetailsSuffix used for getting issue details
	SprintListSuffix = "/rest/greenhopper/1.0/sprintquery/"
)

// IssueDetails contains the logic to use JIRA issues API endpoints
type SprintList struct {
	*Jira
}

// Report wraps Jira Issues API
func (a *Jira) Sprints() (*SprintList, error) {
	return &SprintList{a}, nil
}

// sprintReportURL returns the base URL for the JIRA Sprint Report hidden API
func (a *SprintList) sprintListURL(rapidViewId string) (*url.URL, error) {
	u, err := url.Parse(a.EndpointPrefix)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, SprintListSuffix, rapidViewId)
	return u, nil

}

// Get fetches issue details from Jira API
func (a *SprintList) Get(ctx context.Context, rapidViewId string, includeFutureSprints bool) (*SprintsResponse, error) {

	url, err := a.sprintListURL(rapidViewId)
	if err != nil {
		return nil, err
	}

	// Adding GET parameters
	q := url.Query()
	q.Add("includeFutureSprints", strconv.FormatBool(includeFutureSprints))
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

	var ar SprintsResponse
	err = json.NewDecoder(resp.Body).Decode(&ar)
	if err != nil {
		return nil, err
	}

	return &ar, nil
}
