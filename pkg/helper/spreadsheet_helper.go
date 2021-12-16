package helper

import (
	"context"

	"github.com/jvalecillos/jira-metrics/pkg/googlesheets"
	"google.golang.org/api/sheets/v4"
)

const (
	// How the data is structured
	majorDimension = "ROWS"
	// How the input data should be interpreted.
	valueInputOption = "USER_ENTERED"
	// How the input data should be inserted.
	insertDataOption = "INSERT_ROWS"
)

type SpreadSheetHelper struct {
	srv *sheets.Service
}

func NewSpreadSheetHelper(srv *sheets.Service) SpreadSheetHelper {
	return SpreadSheetHelper{srv: srv}
}

// Append add issues rows to the given spreadsheet and range
func (s SpreadSheetHelper) Append(
	ctx context.Context,
	spreadSheetID string,
	writeRange string,
	dataRows googlesheets.GoogleSheetValues,
) (*sheets.AppendValuesResponse, error) {

	rb := &sheets.ValueRange{
		MajorDimension: majorDimension,
		Values:         dataRows,
	}

	return s.srv.Spreadsheets.Values.Append(spreadSheetID, writeRange, rb).
		ValueInputOption(valueInputOption).
		InsertDataOption(insertDataOption).
		Context(ctx).
		Do()
}

// ResetFormat resets the format from the second row on
func (s SpreadSheetHelper) ResetFormat(
	ctx context.Context,
	spreadSheetID string,
	gid int64,
	startRowIndex int64,
	startColumnIndex int64,
) (*sheets.BatchUpdateSpreadsheetResponse, error) {

	repeatCellRequest := &sheets.RepeatCellRequest{
		Fields: "userEnteredFormat",
		Range: &sheets.GridRange{
			SheetId:          gid,
			StartRowIndex:    startRowIndex,
			StartColumnIndex: startColumnIndex,
		},
		// Cell: &sheets.CellData{
		// 	UserEnteredFormat: &sheets.CellFormat{
		// 		BackgroundColor: &sheets.Color{
		// 			Blue:  1.0,
		// 			Green: 1.0,
		// 			Red:   1.0,
		// 		},
		// 		TextFormat: &sheets.TextFormat{
		// 			ForegroundColor: &sheets.Color{
		// 				Blue:  0.0,
		// 				Green: 0.0,
		// 				Red:   0.0,
		// 			},
		// 		},
		// 	},
		// },
	}

	requestBody := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{{
			RepeatCell: repeatCellRequest,
		}},
	}

	return s.srv.Spreadsheets.
		BatchUpdate(spreadSheetID, requestBody).
		Context(ctx).
		Do()
}
