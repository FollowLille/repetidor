package handlers

import (
	"html/template"
	"math/rand"
	"net/http"
	"path/filepath"
	"repetidor/internal/domain"
	"repetidor/internal/logger"
	"repetidor/internal/storage"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type TrainingHandler struct {
	templates    *template.Template
	topicRepo    storage.TopicRepository
	wordRepo     storage.WordRepository
	trainingRepo storage.TrainingRepository
	logger       logger.Logger
}

type trainingCard struct {
	Word  domain.Word
	Topic domain.Topic
}

func NewTrainingHandler(topicRepo storage.TopicRepository, wordRepo storage.WordRepository, trainingRepo storage.TrainingRepository, appLogger logger.Logger) (*TrainingHandler, error) {
	tmpl, err := template.ParseFiles(filepath.Join("web", "templates", "layout.html"), filepath.Join("web", "templates", "training.html"))
	if err != nil {
		return nil, err
	}
	return &TrainingHandler{templates: tmpl, topicRepo: topicRepo, wordRepo: wordRepo, trainingRepo: trainingRepo, logger: appLogger}, nil
}

func (h *TrainingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.check(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	h.render(w, r, nil)
}

func (h *TrainingHandler) check(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	wordID, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("word_id")), 10, 64)
	if err != nil || wordID <= 0 {
		http.Error(w, "invalid word", http.StatusBadRequest)
		return
	}
	direction := cleanDirection(r.FormValue("direction"))
	prompt := strings.TrimSpace(r.FormValue("prompt"))
	target := strings.TrimSpace(r.FormValue("target"))
	reply := strings.TrimSpace(r.FormValue("reply"))
	ok := domain.NormalizeText(reply) == domain.NormalizeText(target)
	if err := h.trainingRepo.Save(r.Context(), wordID, direction, prompt, target, reply, ok); err != nil {
		h.logger.Error("failed to save training attempt", "error", err, "word_id", wordID)
		http.Error(w, "failed to save training attempt", http.StatusInternalServerError)
		return
	}
	h.render(w, r, map[string]any{"Checked": true, "Correct": ok, "Prompt": prompt, "Target": target, "Reply": reply, "DirectionLabel": labelDirection(direction)})
}

func (h *TrainingHandler) render(w http.ResponseWriter, r *http.Request, result map[string]any) {
	mode := strings.TrimSpace(chi.URLParam(r, "train_mode"))
	if mode == "" {
		mode = "mixed"
	}
	topicID := topicFilter(r)
	direction := directionForMode(mode)
	cards, err := h.cards(r, topicID)
	if err != nil {
		h.logger.Error("failed to load training cards", "error", err)
		http.Error(w, "failed to load training cards", http.StatusInternalServerError)
		return
	}
	progress, err := h.trainingRepo.ListProgress(r.Context(), direction)
	if err != nil {
		h.logger.Error("failed to load training progress", "error", err)
		http.Error(w, "failed to load training progress", http.StatusInternalServerError)
		return
	}
	trainPath := trainingPath(mode, topicID)
	data := map[string]any{"Title": "Training", "PageTitle": "Training", "TrainMode": mode, "TrainPath": trainPath, "Result": result, "NoWords": len(cards) == 0, "PageNotice": "Add words to topics first, then train here."}
	if len(cards) > 0 {
		card := pickCard(cards, progress)
		prompt, target := promptAndTarget(card.Word, direction)
		answerMode := answerModeForMode(mode)
		data["Word"] = card.Word
		data["Topic"] = card.Topic
		data["Direction"] = direction
		data["DirectionLabel"] = labelDirection(direction)
		data["Prompt"] = prompt
		data["Target"] = target
		data["AnswerMode"] = answerMode
		data["Letters"] = shuffledLetters(target)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("failed to render training page", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *TrainingHandler) cards(r *http.Request, topicID int64) ([]trainingCard, error) {
	topics, err := h.topicRepo.List(r.Context())
	if err != nil {
		return nil, err
	}
	cards := make([]trainingCard, 0)
	for _, topic := range topics {
		if topicID > 0 && topic.ID != topicID {
			continue
		}
		words, err := h.wordRepo.ListByTopicID(r.Context(), topic.ID)
		if err != nil {
			return nil, err
		}
		for _, word := range words {
			cards = append(cards, trainingCard{Word: word, Topic: topic})
		}
	}
	return cards, nil
}

func topicFilter(r *http.Request) int64 {
	id, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("topic_id")), 10, 64)
	return id
}

func trainingPath(mode string, topicID int64) string {
	path := "/train/" + mode
	if topicID > 0 {
		path += "?topic_id=" + strconv.FormatInt(topicID, 10)
	}
	return path
}

func pickCard(cards []trainingCard, progress map[int64]domain.TrainingProgress) trainingCard {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	total := 0.0
	weights := make([]float64, len(cards))
	for i, card := range cards {
		weight := trainingWeight(progress[card.Word.ID])
		weights[i] = weight
		total += weight
	}
	roll := r.Float64() * total
	for i, weight := range weights {
		roll -= weight
		if roll <= 0 {
			return cards[i]
		}
	}
	return cards[len(cards)-1]
}

func trainingWeight(progress domain.TrainingProgress) float64 {
	weight := 1.0 + float64(progress.RecentPain)*2.5
	if progress.SeenCount == 0 {
		weight += 2.0
	}
	if progress.CorrectStreak > 0 {
		weight = weight / (1.0 + float64(progress.CorrectStreak)*0.35)
	}
	if weight < 0.25 {
		return 0.25
	}
	return weight
}

func directionForMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "russian-to-spanish", "reverse":
		return "russian_to_spanish"
	case "random", "mixed", "due", "hard", "easy":
		if time.Now().UnixNano()%2 == 0 {
			return "russian_to_spanish"
		}
	}
	return "spanish_to_russian"
}

func answerModeForMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "build", "crossword":
		return "build"
	case "type":
		return "type"
	default:
		if time.Now().UnixNano()%2 == 0 {
			return "build"
		}
		return "type"
	}
}

func shuffledLetters(target string) []string {
	letters := make([]string, 0, len([]rune(target)))
	for _, r := range []rune(strings.ReplaceAll(target, " ", "")) {
		letters = append(letters, string(r))
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(letters), func(i, j int) { letters[i], letters[j] = letters[j], letters[i] })
	return letters
}

func cleanDirection(direction string) string {
	if strings.TrimSpace(direction) == "russian_to_spanish" {
		return "russian_to_spanish"
	}
	return "spanish_to_russian"
}

func labelDirection(direction string) string {
	if direction == "russian_to_spanish" {
		return "Russian to Spanish"
	}
	return "Spanish to Russian"
}

func promptAndTarget(word domain.Word, direction string) (string, string) {
	if direction == "russian_to_spanish" {
		return word.Russian, word.Spanish
	}
	return word.Spanish, word.Russian
}
