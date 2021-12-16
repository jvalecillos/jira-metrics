package googlesheets

import "reflect"

type MySheetRow struct {
	Sprint       string `json:"Sprint"`
	Dicipline    string `json:"Dicipline"`
	TicketNumber string `json:"Ticket Number"`
	Title        string `json:"Title"`
	Link         string `json:"Link"`
	Commited     int    `json:"Commited"`
	Dropped      int    `json:"Dropped"`
	Added        int    `json:"Added"`
	Adjusted     int    `json:"Adjusted"`
	CarriedOver  int    `json:"Carried Over"`
	Completed    int    `json:"Completed"`
}

type MySheetRowArray []MySheetRow

type GoogleSheetValues [][]interface{}

func (m MySheetRowArray) Convert() GoogleSheetValues {

	result := make(GoogleSheetValues, len(m))

	i := 0
	for _, s := range m {

		// transforming the sheet struct in a generic interface array ([]interface{})
		sValue := reflect.ValueOf(s)
		values := make([]interface{}, sValue.NumField())

		for j := 0; j < sValue.NumField(); j++ {
			// copy struct field value into interface
			if sValue.Field(j).CanInterface() {
				values[j] = sValue.Field(j).Interface()
			}
		}

		result[i] = values
		i++
	}

	return result
}
