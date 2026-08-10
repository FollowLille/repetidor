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

	session, err = repo.RecordAnswer(ctx, session.ID, 1, "привет", domain.AnswerEvaluation{Kind: domain.AnswerExact, Correct: true})
	if err != nil {
		t.Fatal(err)
	}
	if session.Completed != 1 || session.Correct != 1 || session.Status != domain.SessionActive {
		t.Fatalf("after first answer = %#v", session)
	}
	if _, err := repo.RecordAnswer(ctx, session.ID, 1, "again", domain.AnswerEvaluation{Kind: domain.AnswerWrong}); err != storage.ErrSessionCardNotFound {
		t.Fatalf("duplicate answer error = %v", err)
	}

	session, err = repo.RecordAnswer(ctx, session.ID, 2, "wrong", domain.AnswerEvaluation{Kind: domain.AnswerTypo, Distance: 1})
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
}
