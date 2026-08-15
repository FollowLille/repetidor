package handlers

import (
	"errors"
	"html/template"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"repetidor/internal/domain"
	"repetidor/internal/game"
	"repetidor/internal/logger"
	"repetidor/internal/storage"

	"github.com/go-chi/chi/v5"
)

type TrainingHandler struct {
	templates    *template.Template
	topicRepo    storage.TopicRepository
	wordRepo     storage.WordRepository
	trainingRepo storage.TrainingRepository
	sessionRepo  storage.SessionRepository
	courseRepo   storage.CourseRepository
	logger       logger.Logger
}

type trainingCard struct {
	Word  domain.Word
	Topic domain.Topic
}
type trainingSessionView struct{ Size, Completed, Current, Correct, Wrong, Skipped int }

func NewTrainingHandler(topicRepo storage.TopicRepository, wordRepo storage.WordRepository, trainingRepo storage.TrainingRepository, sessionRepo storage.SessionRepository, courseRepo storage.CourseRepository, appLogger logger.Logger) (*TrainingHandler, error) {
	tmpl, err := parsePage("training.html")
	if err != nil {
		return nil, err
	}
	return &TrainingHandler{templates: tmpl, topicRepo: topicRepo, wordRepo: wordRepo, trainingRepo: trainingRepo, sessionRepo: sessionRepo, courseRepo: courseRepo, logger: appLogger}, nil
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
	h.render(w, r)
}

func (h *TrainingHandler) check(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	sessionID, err := positiveInt64(r.FormValue("session_id"))
	if err != nil {
		http.Error(w, "invalid session", http.StatusBadRequest)
		return
	}
	if r.FormValue("action") == "retry" {
		position, parseErr := strconv.Atoi(r.FormValue("position"))
		if parseErr != nil || position < 1 {
			http.Error(w, "invalid position", http.StatusBadRequest)
			return
		}
		if err := h.sessionRepo.RequeueCard(r.Context(), sessionID, position); err != nil {
			h.sessionError(w, err)
			return
		}
		session, err := h.sessionRepo.Get(r.Context(), sessionID)
		if err != nil {
			h.sessionError(w, err)
			return
		}
		http.Redirect(w, r, sessionURL(session.Mode, session.ID), http.StatusSeeOther)
		return
	}
	card, err := h.sessionRepo.CurrentCard(r.Context(), sessionID)
	if err != nil {
		h.sessionError(w, err)
		return
	}
	reply := strings.TrimSpace(r.FormValue("reply"))
	action := r.FormValue("action")
	if action == "skip" || action == "dont_know" {
		reply = ""
	}
	prompt, target := promptAndTarget(card.Word, card.Direction)
	evaluation := domain.EvaluateAnswer(reply, target)
	if action == "dont_know" {
		evaluation.Kind = domain.AnswerDontKnow
	}
	if err := h.trainingRepo.SaveSessionAnswer(r.Context(), sessionID, card.Position, card, prompt, target, reply, evaluation); err != nil {
		h.logger.Error("failed to update training session", "error", err, "session_id", sessionID)
		h.sessionError(w, err)
		return
	}
	path := sessionURL(chi.URLParam(r, "train_mode"), sessionID)
	path += "&result_position=" + strconv.Itoa(card.Position)
	http.Redirect(w, r, path, http.StatusSeeOther)
}

func (h *TrainingHandler) render(w http.ResponseWriter, r *http.Request) {
	mode := cleanMode(chi.URLParam(r, "train_mode"))
	sessionID, _ := strconv.ParseInt(r.URL.Query().Get("session_id"), 10, 64)
	if sessionID == 0 {
		session, err := h.createSession(r, mode)
		if errors.Is(err, errNoTrainingCards) {
			h.renderEmpty(w, r, mode)
			return
		}
		if err != nil {
			h.logger.Error("failed to create training session", "error", err)
			http.Error(w, "failed to create training session", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, sessionURL(mode, session.ID), http.StatusSeeOther)
		return
	}
	session, err := h.sessionRepo.Get(r.Context(), sessionID)
	if err != nil {
		h.sessionError(w, err)
		return
	}
	if session.Mode != mode {
		http.Error(w, "session mode mismatch", http.StatusBadRequest)
		return
	}
	data := h.baseData(session)
	data["TrainPath"] = "/train/" + mode
	data["RestartPath"] = trainingPath(mode, session.TopicID)
	if position, _ := strconv.Atoi(r.URL.Query().Get("result_position")); position > 0 {
		card, err := h.sessionRepo.GetCard(r.Context(), sessionID, position)
		if err == nil && card.Answered {
			data["Result"] = resultView(card)
		}
	}
	if session.Status == domain.SessionCompleted {
		data["SessionComplete"] = true
		mistakes, err := h.sessionRepo.MistakeWordIDs(r.Context(), session.ID)
		if err != nil {
			h.logger.Error("failed to load session mistakes", "error", err)
		}
		if len(mistakes) > 0 {
			data["RepeatPath"] = "/train/" + url.PathEscape(mode) + "?repeat_session_id=" + strconv.FormatInt(session.ID, 10)
		}
		h.renderTemplate(w, r, data)
		return
	}
	card, err := h.sessionRepo.CurrentCard(r.Context(), sessionID)
	if err != nil {
		h.sessionError(w, err)
		return
	}
	prompt, _ := promptAndTarget(card.Word, card.Direction)
	data["Card"] = card
	data["Topic"] = card.Topic
	data["Prompt"] = prompt
	data["DirectionLabel"] = labelDirection(card.Direction)
	data["AnswerMode"] = card.AnswerMode
	_, target := promptAndTarget(card.Word, card.Direction)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	data["Letters"] = game.ShuffledLetters(target, rng)
	data["ClozeHint"] = game.MaskWord(target)
	if card.AnswerMode == "choice" || card.AnswerMode == "match" {
		allCards, loadErr := h.cards(r, nil)
		if loadErr != nil {
			h.logger.Error("failed to load arena choices", "error", loadErr)
		} else {
			candidates := make([]string, 0, len(allCards))
			for _, candidate := range allCards {
				_, answer := promptAndTarget(candidate.Word, card.Direction)
				candidates = append(candidates, answer)
			}
			data["Choices"] = game.Choices(target, candidates, game.ChoiceCount(card.AnswerMode), rng)
		}
	}
	h.renderTemplate(w, r, data)
}

var errNoTrainingCards = errors.New("no training cards")

func (h *TrainingHandler) createSession(r *http.Request, mode string) (domain.TrainingSession, error) {
	topicIDs := topicFilters(r)
	topicID := int64(0)
	if len(topicIDs) == 1 {
		topicID = topicIDs[0]
	}
	cards, err := h.cards(r, topicIDs)
	if err != nil {
		return domain.TrainingSession{}, err
	}
	if sourceID, _ := strconv.ParseInt(r.URL.Query().Get("repeat_session_id"), 10, 64); sourceID > 0 {
		ids, err := h.sessionRepo.MistakeWordIDs(r.Context(), sourceID)
		if err != nil {
			return domain.TrainingSession{}, err
		}
		cards = restrictCards(cards, ids)
	}
	directionMode := cleanDirectionMode(r.URL.Query().Get("direction"), mode)
	answerMode := cleanAnswerMode(r.URL.Query().Get("answer"), mode)
	if mode == "arcade" {
		answerMode = strings.Join(arenaGameModes(r.URL.Query()["games"]), ",")
	}
	direction := initialDirection(directionMode, mode)
	spanishProgress, err := h.trainingRepo.ListProgress(r.Context(), "spanish_to_russian")
	if err != nil {
		return domain.TrainingSession{}, err
	}
	russianProgress, err := h.trainingRepo.ListProgress(r.Context(), "russian_to_spanish")
	if err != nil {
		return domain.TrainingSession{}, err
	}
	progressByDirection := map[string]map[int64]domain.TrainingProgress{"spanish_to_russian": spanishProgress, "russian_to_spanish": russianProgress}
	cards = filterCardsForDirections(mode, cards, progressByDirection, time.Now())
	if len(cards) == 0 {
		return domain.TrainingSession{}, errNoTrainingCards
	}
	size := boundedInt(r.URL.Query().Get("session_size"), 10, 1, 50)
	queue := buildSessionQueue(mode, cards, progressByDirection, size, direction, directionMode, answerMode)
	return h.sessionRepo.Create(r.Context(), domain.TrainingSession{Mode: mode, TopicID: topicID, DirectionMode: directionMode, AnswerMode: answerMode}, queue)
}

func buildSessionQueue(mode string, cards []trainingCard, progressByDirection map[string]map[int64]domain.TrainingProgress, size int, firstDirection, directionMode, configuredAnswerMode string) []domain.TrainingSessionCard {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	topicOrder := make([]int64, 0)
	byTopic := make(map[int64][]trainingCard)
	for _, card := range cards {
		if _, exists := byTopic[card.Topic.ID]; !exists {
			topicOrder = append(topicOrder, card.Topic.ID)
		}
		byTopic[card.Topic.ID] = append(byTopic[card.Topic.ID], card)
	}
	queue := make([]domain.TrainingSessionCard, 0, size)
	previousID := int64(0)
	for position := 0; position < size; position++ {
		topicID := topicOrder[position%len(topicOrder)]
		candidates := byTopic[topicID]
		if len(candidates) > 1 {
			candidates = excludeCard(candidates, previousID)
		}
		direction := firstDirection
		if directionMode == "both" && position%2 == 1 {
			direction = oppositeDirection(firstDirection)
		}
		progress := progressByDirection[direction]
		chosen := pickCardForModeWithRand(mode, candidates, progress, rng)
		answerMode := configuredAnswerMode
		if strings.Contains(configuredAnswerMode, ",") {
			playlist := strings.Split(configuredAnswerMode, ",")
			answerMode = playlist[position%len(playlist)]
		}
		if answerMode == "both" {
			if position%2 == 0 {
				answerMode = "type"
			} else {
				answerMode = "build"
			}
		}
		queue = append(queue, domain.TrainingSessionCard{WordID: chosen.Word.ID, TopicID: chosen.Topic.ID, Direction: direction, AnswerMode: answerMode})
		previousID = chosen.Word.ID
	}
	return queue
}

func filterCardsForDirections(mode string, cards []trainingCard, progress map[string]map[int64]domain.TrainingProgress, now time.Time) []trainingCard {
	if mode != "due" && mode != "hard" && mode != "easy" {
		return cards
	}
	left := filterCardsForMode(mode, cards, progress["spanish_to_russian"], now)
	right := filterCardsForMode(mode, cards, progress["russian_to_spanish"], now)
	ids := make(map[int64]bool)
	for _, card := range left {
		ids[card.Word.ID] = true
	}
	for _, card := range right {
		ids[card.Word.ID] = true
	}
	out := make([]trainingCard, 0, len(cards))
	for _, card := range cards {
		if ids[card.Word.ID] {
			out = append(out, card)
		}
	}
	return out
}

func oppositeDirection(direction string) string {
	if direction == "russian_to_spanish" {
		return "spanish_to_russian"
	}
	return "russian_to_spanish"
}

func (h *TrainingHandler) renderEmpty(w http.ResponseWriter, r *http.Request, mode string) {
	size := boundedInt(r.URL.Query().Get("session_size"), 10, 1, 50)
	data := map[string]any{"Title": "Training", "PageTitle": "Training", "TrainMode": mode, "NoWords": true, "PageNotice": emptyModeNotice(mode), "Session": trainingSessionView{Size: size, Current: 1}, "FallbackPath": trainingPath("mixed", topicFilter(r))}
	h.renderTemplate(w, r, data)
}

func (h *TrainingHandler) baseData(session domain.TrainingSession) map[string]any {
	current := session.Completed + 1
	if current > session.Size {
		current = session.Size
	}
	return map[string]any{"Title": modeTitle(session.Mode), "PageTitle": modeTitle(session.Mode), "TrainMode": session.Mode, "SessionID": session.ID, "Session": trainingSessionView{Size: session.Size, Completed: session.Completed, Current: current, Correct: session.Correct, Wrong: session.Completed - session.Correct - session.Skipped, Skipped: session.Skipped}}
}

func resultView(card domain.TrainingSessionCard) map[string]any {
	prompt, target := promptAndTarget(card.Word, card.Direction)
	reply := card.Response
	if reply == "" {
		if card.ErrorKind == domain.AnswerDontKnow {
			reply = "Don't know"
		} else {
			reply = "Skipped"
		}
	}
	feedback := "That answer does not match yet."
	if card.ErrorKind == domain.AnswerTypo {
		feedback = "Very close — this looks like a typo."
	}
	if card.ErrorKind == domain.AnswerSkipped {
		feedback = "Skipped — progress was not changed."
	}
	if card.ErrorKind == domain.AnswerDontKnow {
		feedback = "Marked as unknown — this word will receive more practice."
	}
	return map[string]any{"Correct": card.Correct, "Kind": card.ErrorKind, "Feedback": feedback, "Distance": card.EditDistance, "Position": card.Position, "Prompt": prompt, "Target": target, "Reply": reply, "DirectionLabel": labelDirection(card.Direction)}
}

func (h *TrainingHandler) sessionError(w http.ResponseWriter, err error) {
	if errors.Is(err, storage.ErrSessionNotFound) || errors.Is(err, storage.ErrSessionCardNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if errors.Is(err, storage.ErrSessionComplete) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	h.logger.Error("training session error", "error", err)
	http.Error(w, "training session error", http.StatusInternalServerError)
}

func (h *TrainingHandler) renderTemplate(w http.ResponseWriter, r *http.Request, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", pageData(r, data)); err != nil {
		h.logger.Error("failed to render training page", "error", err)
	}
}

func (h *TrainingHandler) cards(r *http.Request, topicIDs []int64) ([]trainingCard, error) {
	topics, err := h.topicRepo.ListByCourse(r.Context(), activeCourse(h.courseRepo, r).ID)
	if err != nil {
		return nil, err
	}
	var cards []trainingCard
	allowed := make(map[int64]bool)
	for _, id := range topicIDs {
		allowed[id] = true
	}
	for _, topic := range topics {
		if len(allowed) > 0 && !allowed[topic.ID] {
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

func filterCardsForMode(mode string, cards []trainingCard, progress map[int64]domain.TrainingProgress, now time.Time) []trainingCard {
	if mode != "due" && mode != "hard" && mode != "easy" {
		return cards
	}
	filtered := make([]trainingCard, 0, len(cards))
	for _, card := range cards {
		item := progress[card.Word.ID]
		include := mode == "due" && progressIsDue(item, now) || mode == "hard" && item.RecentPain > 0 || mode == "easy" && item.SeenCount > 0 && item.RecentPain == 0 && item.CorrectStreak >= 3
		if include {
			filtered = append(filtered, card)
		}
	}
	return filtered
}

func progressIsDue(p domain.TrainingProgress, now time.Time) bool {
	if p.SeenCount == 0 || p.LastSeenAt == nil || p.RecentPain > 0 {
		return true
	}
	days := []int{0, 1, 3, 7, 14, 30}
	streak := p.CorrectStreak
	if streak >= len(days) {
		streak = len(days) - 1
	}
	return !p.LastSeenAt.Add(time.Duration(days[streak]) * 24 * time.Hour).After(now)
}

func emptyModeNotice(mode string) string {
	switch mode {
	case "due":
		return "Nothing is due right now. Come back later or continue with a mixed session."
	case "hard":
		return "No difficult words right now."
	case "easy":
		return "No mastered words yet. Build a correct streak of at least three."
	default:
		return "Add words to topics first, then train here."
	}
}
func boundedInt(raw string, fallback, min, max int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < min || value > max {
		return fallback
	}
	return value
}
func positiveInt64(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}
func restrictCards(cards []trainingCard, ids []int64) []trainingCard {
	if len(ids) == 0 {
		return nil
	}
	allowed := map[int64]bool{}
	for _, id := range ids {
		allowed[id] = true
	}
	out := make([]trainingCard, 0, len(cards))
	for _, c := range cards {
		if allowed[c.Word.ID] {
			out = append(out, c)
		}
	}
	return out
}
func excludeCard(cards []trainingCard, wordID int64) []trainingCard {
	out := make([]trainingCard, 0, len(cards))
	for _, c := range cards {
		if c.Word.ID != wordID {
			out = append(out, c)
		}
	}
	return out
}
func topicFilter(r *http.Request) int64 {
	id, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("topic_id")), 10, 64)
	return id
}

func topicFilters(r *http.Request) []int64 {
	values := r.URL.Query()["topic_ids"]
	if len(values) == 0 {
		if id := topicFilter(r); id > 0 {
			return []int64{id}
		}
		return nil
	}
	seen := make(map[int64]bool)
	var ids []int64
	for _, raw := range values {
		id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err == nil && id > 0 && !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	return ids
}

func cleanDirectionMode(raw, mode string) string {
	switch raw {
	case "spanish_to_russian", "russian_to_spanish", "both":
		return raw
	}
	if mode == "spanish-to-russian" {
		return "spanish_to_russian"
	}
	if mode == "russian-to-spanish" || mode == "reverse" {
		return "russian_to_spanish"
	}
	return "both"
}

func initialDirection(directionMode, mode string) string {
	if directionMode != "both" {
		return directionMode
	}
	return directionForMode(mode)
}
func cleanAnswerMode(raw, mode string) string {
	switch raw {
	case "type", "build", "both", "choice", "cloze", "anagram", "match":
		return raw
	}
	if mode == "choice" || mode == "match" || mode == "cloze" || mode == "anagram" {
		return mode
	}
	if mode == "arcade" {
		return strings.Join(arenaGameModes(nil), ",")
	}
	if mode == "type" {
		return "type"
	}
	if mode == "build" || mode == "crossword" {
		return "build"
	}
	return "both"
}
func trainingPath(mode string, topicID int64) string {
	path := "/train/" + url.PathEscape(cleanMode(mode))
	if topicID > 0 {
		path += "?topic_id=" + strconv.FormatInt(topicID, 10)
	}
	return path
}
func sessionURL(mode string, id int64) string {
	return "/train/" + url.PathEscape(cleanMode(mode)) + "?session_id=" + strconv.FormatInt(id, 10)
}
func cleanMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "mixed", "random", "type", "build", "crossword", "due", "hard", "easy", "reverse", "russian-to-spanish", "choice", "cloze", "anagram", "match", "arcade":
		return mode
	}
	return "mixed"
}
func pickCard(cards []trainingCard, progress map[int64]domain.TrainingProgress) trainingCard {
	return pickAdaptiveCard(cards, progress, rand.New(rand.NewSource(time.Now().UnixNano())))
}
func pickAdaptiveCard(cards []trainingCard, progress map[int64]domain.TrainingProgress, r *rand.Rand) trainingCard {
	total := 0.0
	weights := make([]float64, len(cards))
	for i, c := range cards {
		weights[i] = trainingWeight(progress[c.Word.ID])
		total += weights[i]
	}
	roll := r.Float64() * total
	for i, w := range weights {
		roll -= w
		if roll <= 0 {
			return cards[i]
		}
	}
	return cards[len(cards)-1]
}
func pickCardForMode(mode string, cards []trainingCard, progress map[int64]domain.TrainingProgress) trainingCard {
	return pickCardForModeWithRand(mode, cards, progress, rand.New(rand.NewSource(time.Now().UnixNano())))
}
func pickCardForModeWithRand(mode string, cards []trainingCard, progress map[int64]domain.TrainingProgress, r *rand.Rand) trainingCard {
	if mode == "random" {
		return cards[r.Intn(len(cards))]
	}
	return pickAdaptiveCard(cards, progress, r)
}
func trainingWeight(p domain.TrainingProgress) float64 {
	weight := 1 + float64(p.RecentPain)*2.5
	if p.SeenCount == 0 {
		weight += 2
	}
	if p.CorrectStreak > 0 {
		weight /= 1 + float64(p.CorrectStreak)*.35
	}
	if weight < .25 {
		return .25
	}
	return weight
}
func directionForMode(mode string) string {
	if mode == "russian-to-spanish" || mode == "reverse" {
		return "russian_to_spanish"
	}
	if (mode == "random" || mode == "mixed") && time.Now().UnixNano()%2 == 0 {
		return "russian_to_spanish"
	}
	return "spanish_to_russian"
}
func answerModeForMode(mode string) string {
	if mode == "choice" || mode == "match" || mode == "cloze" || mode == "anagram" {
		return mode
	}
	if mode == "build" || mode == "crossword" {
		return "build"
	}
	if mode == "type" {
		return "type"
	}
	if time.Now().UnixNano()%2 == 0 {
		return "build"
	}
	return "type"
}

func modeTitle(mode string) string {
	if mode == "arcade" {
		return "Custom arena"
	}
	if item, ok := game.FindMode(mode); ok {
		return item.Name
	}
	return "Training"
}

func arenaGameModes(values []string) []string {
	allowed := map[string]bool{"choice": true, "cloze": true, "anagram": true, "match": true}
	seen := map[string]bool{}
	modes := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if allowed[value] && !seen[value] {
			modes = append(modes, value)
			seen[value] = true
		}
	}
	if len(modes) == 0 {
		return []string{"choice", "cloze", "anagram", "match"}
	}
	return modes
}
func shuffledLetters(target string) []string {
	var letters []string
	for _, r := range []rune(strings.ReplaceAll(target, " ", "")) {
		letters = append(letters, string(r))
	}
	rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(letters), func(i, j int) { letters[i], letters[j] = letters[j], letters[i] })
	return letters
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
