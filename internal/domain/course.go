package domain

import "time"

type Course struct {
	ID                int64
	Name              string
	TargetLanguage    string
	ReferenceLanguage string
	TheoryLanguage    string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Language struct {
	Code, EnglishName, NativeName string
}

var Languages = []Language{
	{Code: "es", EnglishName: "Spanish", NativeName: "Español"},
	{Code: "en", EnglishName: "English", NativeName: "English"},
	{Code: "ru", EnglishName: "Russian", NativeName: "Русский"},
	{Code: "de", EnglishName: "German", NativeName: "Deutsch"},
	{Code: "fr", EnglishName: "French", NativeName: "Français"},
	{Code: "it", EnglishName: "Italian", NativeName: "Italiano"},
	{Code: "pt", EnglishName: "Portuguese", NativeName: "Português"},
}

func LanguageByCode(code string) Language {
	for _, language := range Languages {
		if language.Code == code {
			return language
		}
	}
	return Language{Code: code, EnglishName: code, NativeName: code}
}
