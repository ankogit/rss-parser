package app

import (
	"github.com/k3a/html2text"
	"github.com/mehanizm/airtable"
	"log"
	"strings"
)

func createAirtableRecords(airTable *airtable.Table, records []Item) {

	chunkedRecords := chunkSlice(records, 9)

	for _, chunk := range chunkedRecords {
		recordsToSend := &airtable.Records{
			Records:  []*airtable.Record{},
			Typecast: true,
		}
		for _, item := range chunk {
			country := "-"
			if len(item.Country) > 0 {
				country = item.Country
			}

			var fields = make(map[string]interface{})

			fields["Name"] = item.Title
			fields["Description"] = html2text.HTML2Text(item.Description)
			fields["Resource"] = item.Source
			fields["Status"] = newStatus
			fields["Type"] = item.Type
			fields["Country"] = strings.Title(country)
			fields["URL"] = item.Link
			fields["Tags"] = KeysString(item.Filters)
			fields["Technologies"] = item.Skills

			fields["Budget"] = item.Budget

			if item.Hourly != "" {
				fields["Hourly range"] = item.Hourly
			}
			if item.HourlyFrom != 0 {
				fields["Hour rate from"] = item.HourlyFrom
			}
			if item.HourlyTo != 0 {
				fields["Hour rate to"] = item.HourlyTo
			}

			recordsToSend.Records = append(recordsToSend.Records, &airtable.Record{
				Fields: fields,
			})
		}

		_, err := airTable.AddRecords(recordsToSend)
		if err != nil {
			log.Println(err.Error())
			//log.Panic(err)
		}
	}

}

func chunkSlice(slice []Item, chunkSize int) [][]Item {
	var chunks [][]Item
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		// necessary check to avoid slicing beyond
		// slice capacity
		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}
