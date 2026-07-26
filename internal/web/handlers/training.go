package handlers

import (
	"html/template"
	"math/rand"
	"net/http"
	"net/url"
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

type trainingSession struct {
	Size        int
	Completed   int
	Correct     int
	MistakeIDs  []int64
	OnlyWordIDs []int64
}

type trainingSessionView struct {
	Size      int
	Completed int
	Current   int
	Correct   int
	Wrong     int
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
	h.render(w, r, nil, skippedWord(r), sessionFromRequest(r))
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
	session := sessionFromRequest(r)
	session.Completed++
	if ok {
		session.Correct++
	} else {
		session.MistakeIDs = appendUniqueID(session.MistakeIDs, wordID)
	}
	if err := h.trainingRepo.Save(r.Context(), wordID, direction, prompt, target, reply, ok); err != nil {
		h.logger.Error("failed to save training attempt", "error", err, "word_id", wordID)
		http.Error(w, "failed to save training attempt", http.StatusInternalServerError)
		return
	}
	h.render(w, r, map[string]any{"Checked": true, "Correct": ok, "Prompt": prompt, "Target": target, "Reply": reply, "DirectionLabel": labelDirection(direction)}, wordID, session)
}

func (h *TrainingHandler) render(w http.ResponseWriter, r *http.Request, result map[string]any, excludedWordID int64, session trainingSession) {
	mode := strings.TrimSpace(chi.URLParam(r, "train_mode"))
	if mode == "" {
		mode = "mixed"
	}
	topicID := topicFilter(r)
	sessionView := viewSession(session)
	trainPath := trainingPath(mode, topicID)
	data := map[string]any{
		"Title":        "Training",
		"PageTitle":    "Training",
		"TrainMode":    mode,
		"TrainPath":    trainPath,
		"Result":       result,
		"Session":      sessionView,
		"SessionState": session,
		"RestartPath":  sessionPath(trainPath, trainingSession{Size: session.Size}, 0),
	}
	if session.Completed >= session.Size {
		data["SessionComplete"] = true
		data["RepeatPath"] = repeatMistakesPath(trainPath, session)
		h.renderTemplate(w, r, data)
		return
	}
	direction := directionForMode(mode)
	cards, err := h.cards(r, topicID)
	if err != nil {
		h.logger.Error("failed to load training cards", "error", err)
		http.Error(w, "failed to load training cards", http.StatusInternalServerError)
		return
	}
	cards = restrictCards(cards, session.OnlyWordIDs)
	if len(cards) > 1 && excludedWordID > 0 {
		cards = excludeCard(cards, excludedWordID)
	}
	progress, err := h.trainingRepo.ListProgress(r.Context(), direction)
	if err != nil {
		h.logger.Error("failed to load training progress", "error", err)
		http.Error(w, "failed to load training progress", http.StatusInternalServerError)
		return
	}
	cards = filterCardsForMode(mode, cards, progress, time.Now())
	data["NoWords"] = len(cards) == 0
	data["PageNotice"] = emptyModeNotice(mode)
	data["FallbackPath"] = sessionPath(trainingPath("mixed", topicID), trainingSession{Size: session.Size}, 0)
	if len(cards) > 0 {
		card := pickCardForMode(mode, cards, progress)
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
		data["SkipPath"] = sessionPath(trainPath, session, card.Word.ID)
	}
	h.renderTemplate(w, r, data)
}

func filterCardsForMode(mode string, cards []trainingCard, progress map[int64]domain.TrainingProgress, now time.Time) []trainingCard {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode != "due" && mode != "hard" && mode != "easy" {
		return cards
	}
	filtered := make([]trainingCard, 0, len(cards))
	for _, card := range cards {
		item := progress[card.Word.ID]
		include := false
		switch mode {
		case "due":
			include = progressIsDue(item, now)
		case "hard":
			include = item.RecentPain > 0
		case "easy":
			include = item.SeenCount > 0 && item.RecentPain == 0 && item.CorrectStreak >= 3
		}
		if include {
			filtered = append(filtered, card)
		}
	}
	return filtered
}

func progressIsDue(progress domain.TrainingProgress, now time.Time) bool {
	if progress.SeenCount == 0 || progress.LastSeenAt == nil || progress.RecentPain > 0 {
		return true
	}
	var interval time.Duration
	switch progress.CorrectStreak {
	case 0:
		interval = 0
	case 1:
		interval = 24 * time.Hour
	case 2:
		interval = 3 * 24 * time.Hour
	case 3:
		interval = 7 * 24 * time.Hour
	case 4:
		interval = 14 * 24 * time.Hour
	default:
		interval = 30 * 24 * time.Hour
	}
	return !progress.LastSeenAt.Add(interval).After(now)
}

func emptyModeNotice(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "due":
		return "Nothing is due right now. Come back later or continue with a mixed session."
	case "hard":
		return "No difficult words right now. Wrong answers will appear here until they improve."
	case "easy":
		return "No mastered words yet. Build a correct streak of at least three to unlock this mode."
	default:
		return "Add words to topics first, then train here."
	}
}

func (h *TrainingHandler) renderTemplate(w http.ResponseWriter, r *http.Request, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		h.logger.Error("failed to render training page", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func sessionFromRequest(r *http.Request) trainingSession {
	return trainingSession{
		Size:        boundedInt(r.FormValue("session_size"), 10, 1, 50),
		Completed:   boundedInt(r.FormValue("session_completed"), 0, 0, 50),
		Correct:     boundedInt(r.FormValue("session_correct"), 0, 0, 50),
		MistakeIDs:  parseIDs(r.FormValue("mistake_ids")),
		OnlyWordIDs: parseIDs(r.FormValue("only_word_ids")),
	}
}

func boundedInt(raw string, fallback, min, max int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < min || value > max {
		return fallback
	}
	return value
}

func parseIDs(raw string) []int64 {
	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err == nil && id > 0 {
			ids = appendUniqueID(ids, id)
		}
	}
	return ids
}

func joinIDs(ids []int64) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.FormatInt(id, 10))
	}
	return strings.Join(parts, ",")
}

func appendUniqueID(ids []int64, id int64) []int64 {
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
}

func viewSession(session trainingSession) trainingSessionView {
	current := session.Completed + 1
	if current > session.Size {
		current = session.Size
	}
	return trainingSessionView{
		Size:      session.Size,
		Completed: session.Completed,
		Current:   current,
		Correct:   session.Correct,
		Wrong:     session.Completed - session.Correct,
	}
}

func restrictCards(cards []trainingCard, allowed []int64) []trainingCard {
	if len(allowed) == 0 {
		return cards
	}
	allowedSet := make(map[int64]struct{}, len(allowed))
	for _, id := range allowed {
		allowedSet[id] = struct{}{}
	}
	filtered := make([]trainingCard, 0, len(cards))
	for _, card := range cards {
		if _, ok := allowedSet[card.Word.ID]; ok {
			filtered = append(filtered, card)
		}
	}
	return filtered
}

func sessionPath(base string, session trainingSession, skippedID int64) string {
	parsed, _ := url.Parse(base)
	query := parsed.Query()
	query.Set("session_size", strconv.Itoa(session.Size))
	query.Set("session_completed", strconv.Itoa(session.Completed))
	query.Set("session_correct", strconv.Itoa(session.Correct))
	if mistakes := joinIDs(session.MistakeIDs); mistakes != "" {
		query.Set("mistake_ids", mistakes)
	}
	if only := joinIDs(session.OnlyWordIDs); only != "" {
		query.Set("only_word_ids", only)
	}
	if skippedID > 0 {
		query.Set("skip_word_id", strconv.FormatInt(skippedID, 10))
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func repeatMistakesPath(base string, session trainingSession) string {
	if len(session.MistakeIDs) == 0 {
		return ""
	}
	repeat := trainingSession{Size: len(session.MistakeIDs), OnlyWordIDs: session.MistakeIDs}
	return sessionPath(base, repeat, 0)
}

func skippedWord(r *http.Request) int64 {
	id, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("skip_word_id")), 10, 64)
	return id
}

func excludeCard(cards []trainingCard, wordID int64) []trainingCard {
	filtered := make([]trainingCard, 0, len(cards)-1)
	for _, card := range cards {
		if card.Word.ID != wordID {
			filtered = append(filtered, card)
		}
	}
	return filtered
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

func pickCardForMode(mode string, cards []trainingCard, progress map[int64]domain.TrainingProgress) trainingCard {
	if strings.EqualFold(strings.TrimSpace(mode), "random") {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		return cards[r.Intn(len(cards))]
	}
	return pickCard(cards, progress)
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
	case "random", "mixed":
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
