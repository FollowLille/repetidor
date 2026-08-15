package importx

import (
	"strings"
	"testing"
)

func TestParseTextSupportsUsefulSeparators(t *testing.T) {
	rows := ParseText("hola;привет;greeting\npan\tхлеб\ngracias - спасибо\n# ignored")
	if len(rows) != 3 {
		t.Fatalf("ParseText() got %d rows: %#v", len(rows), rows)
	}
	if rows[0].Source != "hola" || rows[0].Target != "привет" || rows[0].Notes != "greeting" {
		t.Fatalf("first row = %#v", rows[0])
	}
}

func TestParseCSVSkipsHeaderAndReadsNotes(t *testing.T) {
	rows, err := ParseCSV(strings.NewReader("source,target,notes\nhola,привет,greeting\npan,хлеб,"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Notes != "greeting" {
		t.Fatalf("ParseCSV() = %#v", rows)
	}
}

func TestParseSemicolonCSV(t *testing.T) {
	rows, err := ParseCSV(strings.NewReader("spanish;russian;notes\nhola;привет;basic"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Target != "привет" {
		t.Fatalf("ParseCSV() = %#v", rows)
	}
}
