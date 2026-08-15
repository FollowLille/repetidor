package coursepack

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	Format  = "repetidor-course"
	Version = 1
)

type Package struct {
	Format  string `json:"format"`
	Version int    `json:"version"`
	Course  Course `json:"course"`
}

type Course struct {
	Key           string     `json:"key"`
	Name          string     `json:"name"`
	Description   string     `json:"description,omitempty"`
	Target        string     `json:"target_language"`
	Reference     string     `json:"reference_language"`
	Theory        string     `json:"theory_language,omitempty"`
	SortOrder     int        `json:"sort_order,omitempty"`
	ParentKey     string     `json:"parent_key,omitempty"`
	Prerequisites []string   `json:"prerequisite_keys,omitempty"`
	Levels        []string   `json:"levels,omitempty"`
	Blocks        []Block    `json:"theory_blocks,omitempty"`
	Exercises     []Exercise `json:"theory_exercises,omitempty"`
	Topics        []Topic    `json:"topics,omitempty"`
}

type Block struct {
	Key       string `json:"key"`
	Kind      string `json:"kind"`
	Title     string `json:"title,omitempty"`
	Content   string `json:"content"`
	SortOrder int    `json:"sort_order,omitempty"`
}

type Exercise struct {
	Key             string   `json:"key"`
	BlockKey        string   `json:"theory_block_key,omitempty"`
	Kind            string   `json:"kind"`
	Prompt          string   `json:"prompt"`
	CorrectAnswer   string   `json:"correct_answer"`
	Explanation     string   `json:"explanation,omitempty"`
	Options         []string `json:"options,omitempty"`
	AcceptedAnswers []string `json:"accepted_answers,omitempty"`
	SortOrder       int      `json:"sort_order,omitempty"`
}

type Topic struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	SortOrder   int    `json:"sort_order,omitempty"`
	Words       []Word `json:"vocabulary,omitempty"`
}

type Word struct {
	Target    string `json:"target"`
	Reference string `json:"reference"`
	Notes     string `json:"notes,omitempty"`
}

func Decode(data []byte) (Package, error) {
	var value Package
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return Package{}, fmt.Errorf("invalid course JSON: %w", err)
	}
	if err := value.Validate(); err != nil {
		return Package{}, err
	}
	return value, nil
}

func (p Package) Validate() error {
	if p.Format != Format {
		return fmt.Errorf("unsupported format %q", p.Format)
	}
	if p.Version != Version {
		return fmt.Errorf("unsupported course version %d", p.Version)
	}
	if strings.TrimSpace(p.Course.Key) == "" || strings.TrimSpace(p.Course.Name) == "" {
		return errors.New("course key and name are required")
	}
	if p.Course.Target == "" || p.Course.Reference == "" {
		return errors.New("target and reference languages are required")
	}
	keys := map[string]string{}
	add := func(kind, key string) error {
		if key == "" {
			return fmt.Errorf("%s key is required", kind)
		}
		if old := keys[key]; old != "" {
			return fmt.Errorf("duplicate key %q (%s and %s)", key, old, kind)
		}
		keys[key] = kind
		return nil
	}
	blocks := map[string]bool{}
	for _, block := range p.Course.Blocks {
		if err := add("block", block.Key); err != nil {
			return err
		}
		blocks[block.Key] = true
	}
	for _, exercise := range p.Course.Exercises {
		if err := add("exercise", exercise.Key); err != nil {
			return err
		}
		if exercise.Prompt == "" || exercise.CorrectAnswer == "" {
			return fmt.Errorf("exercise %q requires prompt and correct answer", exercise.Key)
		}
		if exercise.BlockKey != "" && !blocks[exercise.BlockKey] {
			return fmt.Errorf("exercise %q references unknown block %q", exercise.Key, exercise.BlockKey)
		}
	}
	for _, topic := range p.Course.Topics {
		if err := add("topic", topic.Key); err != nil {
			return err
		}
		for _, word := range topic.Words {
			if strings.TrimSpace(word.Target) == "" || strings.TrimSpace(word.Reference) == "" {
				return fmt.Errorf("topic %q contains an incomplete word pair", topic.Key)
			}
		}
	}
	return nil
}
