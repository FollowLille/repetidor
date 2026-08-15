package importx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
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

func TestParseExcelReadsFirstNonEmptySheet(t *testing.T) {
	book := excelize.NewFile()
	book.NewSheet("Vocabulary")
	if err := book.SetSheetRow("Vocabulary", "A1", &[]any{"source", "translation", "notes"}); err != nil {
		t.Fatal(err)
	}
	if err := book.SetSheetRow("Vocabulary", "A2", &[]any{"beber", "пить", "verb"}); err != nil {
		t.Fatal(err)
	}
	if err := book.SetSheetRow("Vocabulary", "A3", &[]any{"", "", ""}); err != nil {
		t.Fatal(err)
	}
	if err := book.SetSheetRow("Vocabulary", "A4", &[]any{"comida", "еда"}); err != nil {
		t.Fatal(err)
	}
	buffer, err := book.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := ParseExcel(bytes.NewReader(buffer.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Source != "beber" || rows[0].Notes != "verb" || rows[1].Target != "еда" {
		t.Fatalf("ParseExcel() = %#v", rows)
	}
}

func TestParseExcelRejectsInvalidWorkbook(t *testing.T) {
	if _, err := ParseExcel(strings.NewReader("not an xlsx")); err == nil {
		t.Fatal("ParseExcel() accepted invalid workbook")
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

func TestParseCSVReadsLanguageNeutralVocabularySample(t *testing.T) {
	rows, err := ParseCSV(strings.NewReader("topic,target,reference,notes\nFood,carne,мясо,\nKitchen,cuchara,ложка,feminine\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Topic != "Food" || rows[0].Source != "carne" || rows[0].Target != "мясо" || rows[1].Notes != "feminine" {
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
