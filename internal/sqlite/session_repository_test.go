package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"repetidor/internal/domain"
	"repetidor/internal/storage"
)

func TestSessionRepositoryLifecycle(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "sessions.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	topicResult, err := db.ExecContext(ctx, `INSERT INTO topics(name, description) VALUES('Basics', '')`)
	if err != nil {
		t.Fatal(err)
	}
	topicID, _ := topicResult.LastInsertId()
	wordResult, err := db.ExecContext(ctx, `INSERT INTO words(topic_id, spanish, spanish_key, russian, russian_key, notes) VALUES(?, 'hola', 'hola', 'привет', 'привет', '')`, topicID)
	if err != nil {
		t.Fatal(err)
	}
	wordID, _ := wordResult.LastInsertId()

	repo := NewSessionRepository(db)
	session, err := repo.Create(ctx, domain.TrainingSession{Mode: "mixed", TopicID: topicID}, []domain.TrainingSessionCard{
		{WordID: wordID, TopicID: topicID, Direction: "spanish_to_russian", AnswerMode: "type"},
		{WordID: wordID, TopicID: topicID, Direction: "russian_to_spanish", AnswerMode: "build"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Size != 2 || session.Status != domain.SessionActive {
		t.Fatalf("created session = %#v", session)
	}

	card, err := repo.CurrentCard(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if card.Position != 1 || card.Word.Spanish != "hola" || card.Topic.Name != "Basics" {
		t.Fatalf("first card = %#v", card)
	}

	training := NewTrainingRepository(db)
	if err := training.SaveSessionAnswer(ctx, session.ID, 1, card, "hola", "привет", "привет", domain.AnswerEvaluation{Kind: domain.AnswerExact, Correct: true}); err != nil {
		t.Fatal(err)
	}
	session, err = repo.Get(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if session.Completed != 1 || session.Correct != 1 || session.Status != domain.SessionActive {
		t.Fatalf("after first answer = %#v", session)
	}
	if err := training.SaveSessionAnswer(ctx, session.ID, 1, card, "hola", "привет", "again", domain.AnswerEvaluation{Kind: domain.AnswerWrong}); err != storage.ErrSessionCardNotFound {
		t.Fatalf("duplicate answer error = %v", err)
	}

	second, err := repo.CurrentCard(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := training.SaveSessionAnswer(ctx, session.ID, 2, second, "привет", "hola", "wrong", domain.AnswerEvaluation{Kind: domain.AnswerTypo, Distance: 1}); err != nil {
		t.Fatal(err)
	}
	session, err = repo.Get(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if session.Completed != 2 || session.Correct != 1 || session.Status != domain.SessionCompleted || session.CompletedAt == nil {
		t.Fatalf("completed session = %#v", session)
	}
	mistakes, err := repo.MistakeWordIDs(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mistakes) != 1 || mistakes[0] != wordID {
		t.Fatalf("mistakes = %v", mistakes)
	}
	if err := repo.RequeueCard(ctx, session.ID, 2); err != nil {
		t.Fatal(err)
	}
	reopened, err := repo.Get(ctx, session.ID)
	if err != nil || reopened.Status != domain.SessionActive || reopened.Size != 3 {
		t.Fatalf("reopened = %#v, err = %v", reopened, err)
	}
	retryCard, err := repo.CurrentCard(ctx, session.ID)
	if err != nil || retryCard.Position != 3 || retryCard.WordID != wordID {
		t.Fatalf("retry card = %#v, err = %v", retryCard, err)
	}
	frequent, err := repo.ListFrequentMistakes(ctx, 10)
	if err != nil || len(frequent) != 1 || frequent[0].Typos != 1 {
		t.Fatalf("frequent = %#v, err = %v", frequent, err)
	}
	recent, err := repo.ListRecent(ctx, 10)
	if err != nil || len(recent) != 1 || recent[0].ID != session.ID {
		t.Fatalf("recent = %#v, err = %v", recent, err)
	}
	cards, err := repo.ListCards(ctx, session.ID)
	if err != nil || len(cards) != 3 || !cards[0].Answered || !cards[1].Answered || cards[2].Answered {
		t.Fatalf("cards = %#v, err = %v", cards, err)
	}

	active, err := repo.Create(ctx, domain.TrainingSession{Mode: "type", TopicID: topicID, DirectionMode: "russian_to_spanish", AnswerMode: "type"}, []domain.TrainingSessionCard{{WordID: wordID, TopicID: topicID, Direction: "spanish_to_russian", AnswerMode: "type"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Abandon(ctx, active.ID); err != nil {
		t.Fatal(err)
	}
	abandoned, err := repo.Get(ctx, active.ID)
	if err != nil || abandoned.Status != domain.SessionAbandoned || abandoned.AbandonedAt == nil {
		t.Fatalf("abandoned = %#v, err = %v", abandoned, err)
	}
	if abandoned.DirectionMode != "russian_to_spanish" || abandoned.AnswerMode != "type" {
		t.Fatalf("configuration not persisted: %#v", abandoned)
	}

	skipSession, err := repo.Create(ctx, domain.TrainingSession{Mode: "type", TopicID: topicID}, []domain.TrainingSessionCard{{WordID: wordID, TopicID: topicID, Direction: "spanish_to_russian", AnswerMode: "type"}})
	if err != nil {
		t.Fatal(err)
	}
	skipCard, err := repo.CurrentCard(ctx, skipSession.ID)
	if err != nil {
		t.Fatal(err)
	}
	var attemptsBefore, attemptsAfter, seenBefore, seenAfter int
	if err := db.QueryRow(`SELECT COUNT(*) FROM training_attempts`).Scan(&attemptsBefore); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COALESCE(SUM(seen_count),0) FROM training_progress`).Scan(&seenBefore); err != nil {
		t.Fatal(err)
	}
	if err := training.SaveSessionAnswer(ctx, skipSession.ID, 1, skipCard, "hola", "привет", "", domain.AnswerEvaluation{Kind: domain.AnswerSkipped}); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM training_attempts`).Scan(&attemptsAfter); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COALESCE(SUM(seen_count),0) FROM training_progress`).Scan(&seenAfter); err != nil {
		t.Fatal(err)
	}
	if attemptsAfter != attemptsBefore || seenAfter != seenBefore {
		t.Fatalf("skip changed learning progress: attempts %d->%d seen %d->%d", attemptsBefore, attemptsAfter, seenBefore, seenAfter)
	}

	dontKnowSession, err := repo.Create(ctx, domain.TrainingSession{Mode: "type", TopicID: topicID}, []domain.TrainingSessionCard{{WordID: wordID, TopicID: topicID, Direction: "spanish_to_russian", AnswerMode: "type"}})
	if err != nil {
		t.Fatal(err)
	}
	dontKnowCard, err := repo.CurrentCard(ctx, dontKnowSession.ID)
	if err != nil {
		t.Fatal(err)
	}
	var painBefore, painAfter int
	if err := db.QueryRow(`SELECT recent_pain FROM training_progress WHERE word_id=? AND direction='spanish_to_russian'`, wordID).Scan(&painBefore); err != nil {
		t.Fatal(err)
	}
	if err := training.SaveSessionAnswer(ctx, dontKnowSession.ID, 1, dontKnowCard, "hola", "привет", "", domain.AnswerEvaluation{Kind: domain.AnswerDontKnow}); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT recent_pain FROM training_progress WHERE word_id=? AND direction='spanish_to_russian'`, wordID).Scan(&painAfter); err != nil {
		t.Fatal(err)
	}
	if painAfter != painBefore+2 {
		t.Fatalf("don't know pain %d -> %d", painBefore, painAfter)
	}

	rollbackSession, err := repo.Create(ctx, domain.TrainingSession{Mode: "type", TopicID: topicID}, []domain.TrainingSessionCard{{WordID: wordID, TopicID: topicID, Direction: "spanish_to_russian", AnswerMode: "type"}})
	if err != nil {
		t.Fatal(err)
	}
	rollbackCard, err := repo.CurrentCard(ctx, rollbackSession.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := training.SaveSessionAnswer(ctx, rollbackSession.ID, 2, rollbackCard, "hola", "привет", "wrong", domain.AnswerEvaluation{Kind: domain.AnswerWrong}); err != storage.ErrSessionCardNotFound {
		t.Fatalf("atomic rejection error=%v", err)
	}
	unchanged, err := repo.Get(ctx, rollbackSession.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Completed != 0 {
		t.Fatalf("rejected answer changed session: %#v", unchanged)
	}
}
