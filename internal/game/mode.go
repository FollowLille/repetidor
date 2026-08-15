package game

type Mode struct {
	Slug        string
	Name        string
	Description string
	Icon        string
}

var Modes = []Mode{
	{Slug: "choice", Name: "Quick choice", Description: "Pick the translation from four answers.", Icon: "01"},
	{Slug: "cloze", Name: "Missing letters", Description: "Restore a word from its outline and meaning.", Icon: "Aa"},
	{Slug: "anagram", Name: "Unscramble", Description: "Put shuffled letters back into the right word.", Icon: "↻"},
	{Slug: "match", Name: "Word match", Description: "Find the matching word among a wider field.", Icon: "⌘"},
}

func FindMode(slug string) (Mode, bool) {
	for _, mode := range Modes {
		if mode.Slug == slug {
			return mode, true
		}
	}
	return Mode{}, false
}
