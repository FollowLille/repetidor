package coursepack

import (
	"encoding/json"
	"testing"
)

func TestFormatRoundTripAndBlockLinks(t *testing.T) {
	original := Package{Format: Format, Version: Version, Course: Course{Key: "spanish-a1", Name: "Spanish A1", Target: "es", Reference: "ru", Levels: []string{"A1"}, Blocks: []Block{{Key: "ser", Kind: "text", Title: "Ser", Content: "Rule"}}, Exercises: []Exercise{{Key: "ser-1", BlockKey: "ser", Kind: "input", Prompt: "Yo ___", CorrectAnswer: "soy", AcceptedAnswers: []string{"yo soy"}}}, Topics: []Topic{{Key: "food", Name: "Food", Words: []Word{{Target: "pan", Reference: "хлеб"}}}}}}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Course.Exercises[0].BlockKey != "ser" || decoded.Course.Topics[0].Words[0].Target != "pan" {
		t.Fatalf("links or vocabulary lost: %#v", decoded)
	}
}

func TestFormatRejectsBrokenBlockReference(t *testing.T) {
	value := Package{Format: Format, Version: Version, Course: Course{Key: "x", Name: "X", Target: "es", Reference: "ru", Exercises: []Exercise{{Key: "e", BlockKey: "missing", Prompt: "P", CorrectAnswer: "A"}}}}
	if err := value.Validate(); err == nil {
		t.Fatal("expected broken reference error")
	}
}
