package importx

import (
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

func ParseExcel(reader io.Reader) ([]Row, error) {
	workbook, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("open workbook: %w", err)
	}
	defer workbook.Close()

	for _, sheet := range workbook.GetSheetList() {
		rows, err := workbook.Rows(sheet)
		if err != nil {
			return nil, fmt.Errorf("open sheet %q: %w", sheet, err)
		}
		parsed, err := parseExcelRows(rows)
		if closeErr := rows.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		if err != nil {
			return nil, fmt.Errorf("read sheet %q: %w", sheet, err)
		}
		if len(parsed) > 0 {
			return parsed, nil
		}
	}
	return nil, nil
}

func parseExcelRows(rows *excelize.Rows) ([]Row, error) {
	var result []Row
	line := 0
	firstContentRow := true
	neutral := false
	for rows.Next() {
		line++
		columns, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		for len(columns) < 3 {
			columns = append(columns, "")
		}
		if strings.TrimSpace(strings.Join(columns, "")) == "" {
			continue
		}
		if firstContentRow {
			firstContentRow = false
			neutral = isNeutralHeader(columns)
			if neutral || isHeader(columns) {
				continue
			}
		}
		row := Row{Line: line}
		if neutral {
			for len(columns) < 4 {
				columns = append(columns, "")
			}
			row.Topic = strings.TrimSpace(columns[0])
			row.Source = strings.TrimSpace(columns[1])
			row.Target = strings.TrimSpace(columns[2])
			row.Notes = strings.TrimSpace(columns[3])
		} else {
			row.Source = strings.TrimSpace(columns[0])
			row.Target = strings.TrimSpace(columns[1])
			row.Notes = strings.TrimSpace(columns[2])
		}
		if row.Source != "" && row.Target != "" {
			result = append(result, row)
		}
	}
	return result, rows.Error()
}
