package importx

import (
	"encoding/csv"
	"io"
	"strings"
)

func ParseCSV(reader io.Reader) ([]Row, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	content := string(data)
	delimiter := ','
	first := content
	if i := strings.IndexByte(content, '\n'); i >= 0 {
		first = content[:i]
	}
	if strings.Count(first, ";") > strings.Count(first, ",") {
		delimiter = ';'
	}
	parser := csv.NewReader(strings.NewReader(content))
	parser.Comma = delimiter
	parser.FieldsPerRecord = -1
	parser.TrimLeadingSpace = true
	var rows []Row
	line := 0
	for {
		record, err := parser.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		line++
		if len(record) < 2 {
			continue
		}
		if line == 1 && isHeader(record) {
			continue
		}
		row := Row{Source: strings.TrimSpace(record[0]), Target: strings.TrimSpace(record[1]), Line: line}
		if len(record) > 2 {
			row.Notes = strings.TrimSpace(record[2])
		}
		if row.Source != "" && row.Target != "" {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func isHeader(record []string) bool {
	left := strings.ToLower(strings.TrimSpace(record[0]))
	right := strings.ToLower(strings.TrimSpace(record[1]))
	return (left == "source" || left == "word" || left == "spanish" || left == "слово") && (right == "target" || right == "translation" || right == "russian" || right == "перевод")
}
