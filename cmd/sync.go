package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/jvalecillos/jira-metrics/pkg/googlesheets"
	"github.com/jvalecillos/jira-metrics/pkg/helper"
	"github.com/jvalecillos/jira-metrics/pkg/jira"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type serviceWrapper struct {
	context            context.Context
	jiraClient         *jira.Jira
	spreadSheetsHelper helper.SpreadSheetHelper
}

var all bool
var year string
var jiraProject string
var sv *serviceWrapper

type sprint struct {
	id   string
	name string
}

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Syncs Sprints from JIRA to GoogleSheet",
	Long: `Fetches the Sprint information from JIRA and syncs it with
the given GoogleSheet.

Example: jira-metrics sync --year 2021 [--all | --sprint-week 41-43]`,
	PreRunE: func(cmd *cobra.Command, args []string) error {

		ctx := context.Background()

		// https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/
		jc, err := jira.New(jira.Config{
			Username:       viper.GetString("JIRA_USERNAME"),
			Password:       viper.GetString("JIRA_TOKEN"),
			EndpointPrefix: viper.GetString("JIRA_ENDPOINT_PREFIX"),
		}, nil)

		if err != nil {
			return errors.Wrap(err, "error creating JIRA client")
		}

		googleSheetsSrv, err := googlesheets.NewService(ctx,
			"credentials.json",
			// scope for reading
			// "https://www.googleapis.com/auth/spreadsheets.readonly",
			// scope to edit only an specific sheet
			// "https://www.googleapis.com/auth/drive.file",
			// scope for writing all sheets
			"https://www.googleapis.com/auth/spreadsheets",
		)

		if err != nil {
			return errors.Wrap(err, "error initializing Google Sheets service")
		}

		sv = &serviceWrapper{
			context:            ctx,
			jiraClient:         jc,
			spreadSheetsHelper: helper.NewSpreadSheetHelper(googleSheetsSrv),
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		SprintListSrv, _ := sv.jiraClient.Sprints()

		fmt.Printf("Fetching Sprints from project %s...\n", jiraProject)

		sprintList, err := SprintListSrv.Get(sv.context, jiraProject, false)
		if err != nil {
			return errors.Wrap(err, "error getting Sprint list")
		}

		fmt.Printf("Filtering Sprints from year %s...\n", year)

		// Pattern for filtering Sprints
		// @TODO: Should this be part of the configuration?
		r, err := regexp.Compile(fmt.Sprintf(`(?:[A-Z]{2,3})\s+Sprint\s+(%s)[-\s]?W?(\d{2})-W?(\d{2})`, year))

		if err != nil {
			return errors.Wrap(err, "error comiling Sprint regex")
		}

		var sprintLookupMap map[string]string = make(map[string]string, len(sprintList.Sprints))
		var orderedSprintList []sprint = nil
		var sprintPromptOptions []string = []string{}

		for _, s := range sprintList.Sprints {
			// filtering non-closed Sprint
			if s.State != "CLOSED" {
				continue
			}
			// filtering relevant Sprints by name pattern
			if !r.MatchString(s.Name) && s.Name != "STR Sprint W51-W02(2021-2022)" {
				continue
			}
			sprintLookupMap[s.Name] = strconv.Itoa(s.ID)
			orderedSprintList = append(orderedSprintList, sprint{id: strconv.Itoa(s.ID), name: s.Name})
			sprintPromptOptions = append(sprintPromptOptions, s.Name)
		}

		// Syncing the whole year
		if all {
			fmt.Printf("Syncing all Sprints for %s...\n", year)
			if err := sv.syncAll(orderedSprintList); err != nil {
				return errors.Wrap(err, "error syncing ALL Sprint")
			}
		} else {
			// Syncing a single Sprint
			var selectedSprint string

			prompt := &survey.Select{
				Message: "Choose a Sprint:",
				Options: sprintPromptOptions,
			}
			survey.AskOne(prompt, &selectedSprint)

			sprintID := sprintLookupMap[selectedSprint]

			if err := sv.syncSprint(sprintID, selectedSprint); err != nil {
				return errors.Wrap(err, "error syncing Sprint")
			}
		}

		// Format resetting
		spreadSheetID := viper.GetString("GOOGLE_SPREADSHEET")
		issuesGid := viper.GetInt64("GOOGLE_SPREADSHEET_TICKETS_GID")
		sprintListGid := viper.GetInt64("GOOGLE_SPREADSHEET_SPRINTS_GID")

		if err := sv.resetIssuesFormat(spreadSheetID, issuesGid); err != nil {
			return err
		}

		if err := sv.resetSprintListFormat(spreadSheetID, sprintListGid); err != nil {
			return err
		}

		fmt.Printf("ALL DONE!\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// flags and configuration settings.
	syncCmd.Flags().StringVarP(&jiraProject, "project", "p", "", "Project ID from JIRA (required)")
	syncCmd.MarkFlagRequired("project")
	syncCmd.Flags().StringVarP(&year, "year", "y", "2021", "Year for filtering Sprints (required)")
	syncCmd.MarkFlagRequired("year")
	syncCmd.Flags().BoolVarP(&all, "all", "a", false, "Sync ALL Sprints in the year")
}

// syncAll syncs all the Sprint from a list to the Google Spreadsheet
func (sv serviceWrapper) syncAll(sprints []sprint) error {

	for _, sprint := range sprints {
		if err := sv.syncSprint(sprint.id, sprint.name); err != nil {
			return errors.Wrapf(err, "error syncing Sprint %s", sprint.name)
		}
	}

	return nil
}

// syncSprint syncs a single Sprint to the Google Spreadsheet
func (sv serviceWrapper) syncSprint(sprintID, sprintName string) error {

	sprintReportSrv, _ := sv.jiraClient.Report()

	sprintReport, err := sprintReportSrv.Get(sv.context, jiraProject, sprintID)
	if err != nil {
		return errors.Wrap(err, "error getting Sprint report")
	}

	issuesSrv, _ := sv.jiraClient.Issues()

	issuesHelper := helper.NewIssuesHelper(issuesSrv)

	fmt.Printf("Processing report for %s...\n", sprintName)

	allIssues, err := issuesHelper.ProcessReport(*sprintReport)
	if err != nil {
		return errors.Wrap(err, "error processing Sprint report")
	}

	fmt.Printf("Writing issues for %s in Google Sheets...\n", sprintName)

	if _, err := sv.spreadSheetsHelper.Append(
		sv.context,
		viper.GetString("GOOGLE_SPREADSHEET"),
		viper.GetString("GOOGLE_SPREADSHEET_TICKETS_WR"),
		allIssues.Convert(),
	); err != nil {
		return errors.Wrap(err, "error writing issues in GoogleSheets")
	}

	fmt.Printf("Adding Sprint to list %s in Google Sheets...\n", sprintName)

	var sprintRows googlesheets.GoogleSheetValues = [][]interface{}{
		{sprintName, sprintID, helper.SimplifySprintName(sprintName)},
	}

	if err := sv.addSprintsToList(sprintRows); err != nil {
		return errors.Wrapf(err, "errors adding Sprint %s to list", sprintName)
	}

	return nil
}

// addSprintsToList adds a list of Sprints to a sheet in the Google Spreadsheet
func (sv serviceWrapper) addSprintsToList(sprintRows googlesheets.GoogleSheetValues) error {

	fmt.Printf("Adding Sprints to list in Google Sheets...\n")

	if _, err := sv.spreadSheetsHelper.Append(
		sv.context,
		viper.GetString("GOOGLE_SPREADSHEET"),
		viper.GetString("GOOGLE_SPREADSHEET_SPRINTS_WR"),
		sprintRows,
	); err != nil {
		return errors.Wrap(err, "error adding Sprints to Sprint list Google Sheets")
	}

	return nil
}

// resetIssuesFormat sets the default style for rows of JIRA issues
func (sv serviceWrapper) resetIssuesFormat(spreadSheetID string, gid int64) error {
	fmt.Printf("Resetting format for issues list in Google Sheets...\n")

	if _, err := sv.spreadSheetsHelper.ResetFormat(sv.context, spreadSheetID, gid, 1, 0); err != nil {
		return errors.Wrap(err, "error resetting issues format in Google Sheets")
	}
	return nil
}

// resetSprintListFormat sets the default style for rows of the Sprint list
func (sv serviceWrapper) resetSprintListFormat(spreadSheetID string, gid int64) error {
	fmt.Printf("Resetting format for Sprint list in Google Sheets...\n")

	if _, err := sv.spreadSheetsHelper.ResetFormat(sv.context, spreadSheetID, gid, 1, 0); err != nil {
		return errors.Wrap(err, "error resetting Sprint list format in Google Sheets")
	}
	return nil
}
