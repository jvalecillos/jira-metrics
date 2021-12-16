package helper

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/jvalecillos/jira-metrics/pkg/googlesheets"
	"github.com/jvalecillos/jira-metrics/pkg/jira"
)

type IssuesHelper struct {
	srv *jira.IssueDetails
}

func NewIssuesHelper(srv *jira.IssueDetails) IssuesHelper {
	return IssuesHelper{srv: srv}
}

func (d IssuesHelper) ProcessReport(report jira.ReportResponse) (googlesheets.MySheetRowArray, error) {

	var rowArray googlesheets.MySheetRowArray = make(
		googlesheets.MySheetRowArray,
		len(report.Contents.CompletedIssues)+
			len(report.Contents.IssuesNotCompletedInCurrentSprint)+
			len(report.Contents.PuntedIssues),
	)

	index := 0

	// fillup completed tickets
	for _, j := range report.Contents.CompletedIssues {

		added := false
		// checking if the issue was added to the Sprint
		if _, ok := report.Contents.IssueKeysAddedDuringSprint[j.Key]; ok {
			added = true
		}

		rowArray[index] = d.generateRow(j, added, report.Sprint.Name, issueCompleted)
		index++
	}

	// fillup not-completed tickets
	for _, j := range report.Contents.IssuesNotCompletedInCurrentSprint {

		added := false
		// checking if the issue was added to the Sprint
		if _, ok := report.Contents.IssueKeysAddedDuringSprint[j.Key]; ok {
			added = true
		}

		rowArray[index] = d.generateRow(j, added, report.Sprint.Name, issueNotCompleted)
		index++
	}

	// fillup removed from the Sprint
	for _, j := range report.Contents.PuntedIssues {

		added := false
		// checking if the issue was added to the Sprint
		if _, ok := report.Contents.IssueKeysAddedDuringSprint[j.Key]; ok {
			added = true
		}

		rowArray[index] = d.generateRow(j, added, report.Sprint.Name, issueRemoved)
		index++
	}

	return rowArray, nil
}

const (
	issueCompleted    = "completed"
	issueNotCompleted = "notCompleted"
	issueRemoved      = "removed"
)

func (i IssuesHelper) generateRow(
	j jira.Issue,
	added bool,
	sprintName string,
	issueCategory string,
) googlesheets.MySheetRow {
	row := googlesheets.MySheetRow{
		Sprint:       SimplifySprintName(sprintName),
		TicketNumber: j.Key,
		Title:        j.Summary,
		Link:         i.generateJiraLink(j.Key, j.Summary),
	}

	// If the ticket was added after starting the Sprint
	if added {
		// original estimation
		row.Added = int(j.EstimateStatistic.StatFieldValue.Value)
	} else {
		// original estimation
		row.Commited = int(j.EstimateStatistic.StatFieldValue.Value)
	}

	if issueCategory == issueCompleted {
		// original estimation
		row.Completed = int(j.EstimateStatistic.StatFieldValue.Value)
	}

	if issueCategory == issueNotCompleted {
		// new estimation
		row.CarriedOver = int(j.CurrentEstimateStatistic.StatFieldValue.Value)
	}

	if issueCategory == issueRemoved {
		// original estimation
		row.Dropped = int(j.EstimateStatistic.StatFieldValue.Value)
	}

	row.Adjusted = row.Commited - row.Dropped + row.Added

	row.Dicipline, _ = i.solveDicipline(j)

	return row
}

const issueBrowseSuffix = "/browse/"

// generateJiraLink generates a GoogleSheet HyperLink given the issueID and the title
func (i IssuesHelper) generateJiraLink(issueID, issueTitle string) string {
	// Parse endpoint prefix URL
	u, _ := url.Parse(i.srv.EndpointPrefix)
	u.Path = path.Join(u.Path, issueBrowseSuffix, issueID)

	return fmt.Sprintf(
		"=HYPERLINK(\"%s\",\"%s\")",
		u.String(),
		strings.TrimSpace(strings.ReplaceAll(issueTitle, "\"", "'")),
	)
}

var sprintRegex = regexp.MustCompile(`(?:IMR|MNZ|STR)\s+Sprint\s+(\d{4})-W?(\d{2})-W?(\d{2})`)

// SimplifySprintName changes the Sprint name to YYYY-WNN-NN
func SimplifySprintName(name string) string {
	// no match
	if !sprintRegex.MatchString(name) {
		return name
	}
	matches := sprintRegex.FindStringSubmatch(name)
	// no matches with the expected groups
	if len(matches) != 4 {
		return name
	}
	return fmt.Sprintf("%s-W%s-%s", matches[1], matches[2], matches[3])
}

// regexMap is a list of regular expressions for looking up the dicipline in the title
var regexMap = map[string]*regexp.Regexp{
	"Backend": regexp.MustCompile(`(?i).*\[\s*(?:[^Web|AutoQA|Android|iOS])?\s*(Backend).*\].*`),
	"Web":     regexp.MustCompile(`(?i).*\[\s*(?:[^Backend|AutoQA|Android|iOS])?\s*(Web).*\].*`),
	"AutoQA":  regexp.MustCompile(`(?i).*\[\s*(?:[^Backend|Web|Android|iOS])?\s*(AutoQA).*\].*`),
	"Android": regexp.MustCompile(`(?i).*\[\s*(?:[^Backend|Web|AutoQA|iOS])?\s*(Android).*\].*`),
	"iOS":     regexp.MustCompile(`(?i).*\[\s*(?:[^Backend|Web|AutoQA|Android])?\s*(iOS).*\].*`),
}

// disciplines caches disciplines for already checked issues
var disciplines map[string]string = make(map[string]string)

// solveDicipline tries to find the dicipline for a given issue
func (i IssuesHelper) solveDicipline(issue jira.Issue) (string, error) {

	// check in local cache
	if d, ok := disciplines[issue.Key]; ok {
		fmt.Printf("discipline %s found in cache for %s\n", d, issue.Key)
		return d, nil
	}

	for key, reg := range regexMap {
		if reg.MatchString(issue.Summary) {
			// save discipline for next lookup
			disciplines[issue.Key] = key
			return key, nil
		}
	}

	issueWithDetails, err := i.srv.Get(context.Background(), issue.Key)
	if err != nil {
		return "Other", err
	}

	if issueWithDetails.Fields.Discipline.Value != "" {
		// extract discipline from custom field
		// save discipline for next lookup
		disciplines[issue.Key] = issueWithDetails.Fields.Discipline.Value
		return issueWithDetails.Fields.Discipline.Value, nil
	} else {
		if len(issueWithDetails.Fields.Components) > 0 {
			// extract discipline from first component
			// save discipline for next lookup
			disciplines[issue.Key] = issueWithDetails.Fields.Components[0].Name
			return issueWithDetails.Fields.Components[0].Name, nil
		}
	}

	return "Other", nil
}
