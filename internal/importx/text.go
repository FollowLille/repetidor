package importx

import (
	"bufio"
	"strings"
)

type Row struct {
	Source, Target, Notes string
	Line                  int
}

func ParseText(input string) []Row {
	var rows []Row
	scanner := bufio.NewScanner(strings.NewReader(input))
	line := 0
	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		parts := splitLine(raw)
		if len(parts) < 2 {
			continue
		}
		row := Row{Source: strings.TrimSpace(parts[0]), Target: strings.TrimSpace(parts[1]), Line: line}
		if len(parts) > 2 {
			row.Notes = strings.TrimSpace(strings.Join(parts[2:], " "))
		}
		if row.Source != "" && row.Target != "" {
			rows = append(rows, row)
		}
	}
	return rows
}

func splitLine(line string) []string {
	for _, separator := range []string{"\t", ";", " → ", " -> ", " - "} {
		if strings.Contains(line, separator) {
			return strings.Split(line, separator)
		}
	}
	return strings.Split(line, ",")
}
