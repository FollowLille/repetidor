package game

func ChoiceCount(mode string) int {
	if mode == "match" {
		return 6
	}
	return 4
}
