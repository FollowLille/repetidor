package sqlite

import (
	"context"
	"errors"
	"os"
	"testing"

	"repetidor/internal/domain"
	"repetidor/internal/storage"
)

func TestRepositoriesMapUniqueConstraintErrors(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		CREATE TABLE topics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			language_track_id INTEGER NOT NULL DEFAULT 1,
			name TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE words (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			topic_id INTEGER NOT NULL,
			spanish TEXT NOT NULL,
			spanish_key TEXT NOT NULL,
			russian TEXT NOT NULL,
			russian_key TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(topic_id, spanish_key, russian_key)
		);
		CREATE TABLE word_topics (
			word_id INTEGER NOT NULL,
			topic_id INTEGER NOT NULL,
			PRIMARY KEY(word_id, topic_id)
		);
	`)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	topics := NewTopicRepository(db)
	topic, err := topics.Create(ctx, domain.Topic{Name: "Travel"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := topics.Create(ctx, domain.Topic{Name: "Travel"}); !errors.Is(err, storage.ErrTopicAlreadyExists) {
		t.Fatalf("duplicate topic error = %v, want ErrTopicAlreadyExists", err)
	}

	second, err := topics.Create(ctx, domain.Topic{Name: "Food"})
	if err != nil {
		t.Fatal(err)
	}
	second.Name = topic.Name
	if _, err := topics.Update(ctx, second); !errors.Is(err, storage.ErrTopicAlreadyExists) {
		t.Fatalf("duplicate topic update error = %v, want ErrTopicAlreadyExists", err)
	}

	words := NewWordRepository(db)
	word := domain.Word{TopicID: topic.ID, Spanish: "hola", Russian: "привет"}
	if _, err := words.Create(ctx, word); err != nil {
		t.Fatal(err)
	}
	word.Spanish = " HOLA "
	if _, err := words.Create(ctx, word); !errors.Is(err, storage.ErrWordAlreadyExists) {
		t.Fatalf("duplicate word error = %v, want ErrWordAlreadyExists", err)
	}
}

func TestWordRepositoryDeleteIsScopedToTopic(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		CREATE TABLE words (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			topic_id INTEGER NOT NULL,
			spanish TEXT NOT NULL,
			spanish_key TEXT NOT NULL,
			russian TEXT NOT NULL,
			russian_key TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(topic_id, spanish_key, russian_key)
		);
		CREATE TABLE word_topics (
			word_id INTEGER NOT NULL,
			topic_id INTEGER NOT NULL,
			PRIMARY KEY(word_id, topic_id)
		);
	`)
	if err != nil {
		t.Fatal(err)
	}

	repo := NewWordRepository(db)
	word, err := repo.Create(context.Background(), domain.Word{TopicID: 1, Spanish: "hola", Russian: "привет"})
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.Delete(context.Background(), 2, word.ID); !errors.Is(err, storage.ErrWordNotFound) {
		t.Fatalf("delete from another topic error = %v, want ErrWordNotFound", err)
	}
	if err := repo.Delete(context.Background(), 1, word.ID); err != nil {
		t.Fatalf("delete word: %v", err)
	}
	if err := repo.Delete(context.Background(), 1, word.ID); !errors.Is(err, storage.ErrWordNotFound) {
		t.Fatalf("delete missing word error = %v, want ErrWordNotFound", err)
	}
}

func TestTopicRepositoryDeleteCascadesToWords(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		CREATE TABLE topics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			language_track_id INTEGER NOT NULL DEFAULT 1,
			name TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE words (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			topic_id INTEGER NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
			spanish TEXT NOT NULL,
			spanish_key TEXT NOT NULL,
			russian TEXT NOT NULL,
			russian_key TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(topic_id, spanish_key, russian_key)
		);
		CREATE TABLE word_topics (
			word_id INTEGER NOT NULL REFERENCES words(id) ON DELETE CASCADE,
			topic_id INTEGER NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
			PRIMARY KEY(word_id, topic_id)
		);
	`)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	topics := NewTopicRepository(db)
	topic, err := topics.Create(ctx, domain.Topic{Name: "Travel"})
	if err != nil {
		t.Fatal(err)
	}
	words := NewWordRepository(db)
	if _, err := words.Create(ctx, domain.Word{TopicID: topic.ID, Spanish: "viaje", Russian: "путешествие"}); err != nil {
		t.Fatal(err)
	}

	if err := topics.Delete(ctx, topic.ID); err != nil {
		t.Fatalf("delete topic: %v", err)
	}
	var wordCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM words WHERE topic_id = ?`, topic.ID).Scan(&wordCount); err != nil {
		t.Fatal(err)
	}
	if wordCount != 0 {
		t.Fatalf("words remaining after topic deletion = %d, want 0", wordCount)
	}
	if err := topics.Delete(ctx, topic.ID); !errors.Is(err, storage.ErrTopicNotFound) {
		t.Fatalf("delete missing topic error = %v, want ErrTopicNotFound", err)
	}
}

func TestWordRepositoryUpdateIsScopedAndMapsDuplicates(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`
		CREATE TABLE words (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			topic_id INTEGER NOT NULL,
			spanish TEXT NOT NULL,
			spanish_key TEXT NOT NULL,
			russian TEXT NOT NULL,
			russian_key TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(topic_id, spanish_key, russian_key)
		);
		CREATE TABLE word_topics (
			word_id INTEGER NOT NULL,
			topic_id INTEGER NOT NULL,
			PRIMARY KEY(word_id, topic_id)
		);
	`)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	repo := NewWordRepository(db)
	first, err := repo.Create(ctx, domain.Word{TopicID: 1, Spanish: "hola", Russian: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.Create(ctx, domain.Word{TopicID: 1, Spanish: "adios", Russian: "goodbye"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByID(ctx, 2, first.ID); !errors.Is(err, storage.ErrWordNotFound) {
		t.Fatalf("get word from another topic error = %v, want ErrWordNotFound", err)
	}

	first.Spanish = "buenos dias"
	first.Russian = "good morning"
	first.Notes = "greeting"
	updated, err := repo.Update(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Spanish != first.Spanish || updated.Notes != first.Notes {
		t.Fatalf("updated word = %#v", updated)
	}

	updated.Spanish = second.Spanish
	updated.Russian = second.Russian
	if _, err := repo.Update(ctx, updated); !errors.Is(err, storage.ErrWordAlreadyExists) {
		t.Fatalf("duplicate update error = %v, want ErrWordAlreadyExists", err)
	}
}

func TestTrainingRepositoryListStatsIncludesSeenAndUnseenWords(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`
		CREATE TABLE topics (id INTEGER PRIMARY KEY, language_track_id INTEGER NOT NULL DEFAULT 1, name TEXT NOT NULL);
		CREATE TABLE words (
			id INTEGER PRIMARY KEY,
			topic_id INTEGER NOT NULL,
			spanish TEXT NOT NULL,
			spanish_key TEXT NOT NULL,
			russian TEXT NOT NULL,
			russian_key TEXT NOT NULL
		);
		CREATE TABLE training_attempts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			word_id INTEGER NOT NULL,
			direction TEXT NOT NULL,
			question TEXT NOT NULL,
			expected TEXT NOT NULL,
			response TEXT NOT NULL,
			is_correct INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE training_progress (
			word_id INTEGER NOT NULL,
			direction TEXT NOT NULL,
			seen_count INTEGER NOT NULL DEFAULT 0,
			correct_count INTEGER NOT NULL DEFAULT 0,
			wrong_count INTEGER NOT NULL DEFAULT 0,
			correct_streak INTEGER NOT NULL DEFAULT 0,
			recent_pain INTEGER NOT NULL DEFAULT 0,
			last_seen_at DATETIME,
			last_correct_at DATETIME,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (word_id, direction)
		);
		CREATE TABLE word_topics (
			word_id INTEGER NOT NULL,
			topic_id INTEGER NOT NULL,
			PRIMARY KEY(word_id, topic_id)
		);
		INSERT INTO topics(id, name) VALUES(1, 'Travel');
		INSERT INTO words(id, topic_id, spanish, spanish_key, russian, russian_key) VALUES
			(1, 1, 'viaje', 'viaje', 'journey', 'journey'),
			(2, 1, 'tren', 'tren', 'train', 'train');
		INSERT INTO word_topics(word_id, topic_id) VALUES(1, 1), (2, 1);
	`)
	if err != nil {
		t.Fatal(err)
	}

	repo := NewTrainingRepository(db)
	ctx := context.Background()
	if err := repo.Save(ctx, 1, "spanish_to_russian", "viaje", "journey", "journey", true); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, 1, "spanish_to_russian", "viaje", "journey", "trip", false); err != nil {
		t.Fatal(err)
	}
	progress, err := repo.ListProgress(ctx, "spanish_to_russian")
	if err != nil {
		t.Fatal(err)
	}
	if progress[1].LastSeenAt == nil {
		t.Fatal("last seen time was not loaded")
	}
	stats, err := repo.ListStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 2 {
		t.Fatalf("stats length = %d, want 2", len(stats))
	}
	if stats[0].WordID != 1 || stats[0].SeenCount != 2 || stats[0].CorrectCount != 1 || stats[0].WrongCount != 1 || stats[0].Accuracy != 50 || stats[0].RecentPain != 2 {
		t.Fatalf("seen word stats = %#v", stats[0])
	}
	if stats[1].WordID != 2 || stats[1].SeenCount != 0 {
		t.Fatalf("unseen word stats = %#v", stats[1])
	}
}

func TestWordCanBelongToMultipleTopics(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`
		CREATE TABLE topics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			language_track_id INTEGER NOT NULL DEFAULT 1,
			name TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE words (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			topic_id INTEGER NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
			spanish TEXT NOT NULL,
			spanish_key TEXT NOT NULL,
			russian TEXT NOT NULL,
			russian_key TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(topic_id, spanish_key, russian_key)
		);
		CREATE TABLE word_topics (
			word_id INTEGER NOT NULL REFERENCES words(id) ON DELETE CASCADE,
			topic_id INTEGER NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
			PRIMARY KEY(word_id, topic_id)
		);
	`)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	topics := NewTopicRepository(db)
	travel, err := topics.Create(ctx, domain.Topic{Name: "Travel"})
	if err != nil {
		t.Fatal(err)
	}
	favorites, err := topics.Create(ctx, domain.Topic{Name: "Favorites"})
	if err != nil {
		t.Fatal(err)
	}
	words := NewWordRepository(db)
	first, err := words.Create(ctx, domain.Word{TopicID: travel.ID, Spanish: "viaje", Russian: "journey", Notes: "shared"})
	if err != nil {
		t.Fatal(err)
	}
	shared, err := words.Create(ctx, domain.Word{TopicID: favorites.ID, Spanish: " VIAJE ", Russian: "journey"})
	if err != nil {
		t.Fatal(err)
	}
	if shared.ID != first.ID {
		t.Fatalf("shared word id = %d, want %d", shared.ID, first.ID)
	}
	updated, err := words.Update(ctx, domain.Word{ID: first.ID, TopicID: favorites.ID, Spanish: "viaje", Russian: "trip", Notes: "edited from favorites"})
	if err != nil {
		t.Fatalf("edit shared word from secondary topic: %v", err)
	}
	if updated.TopicID != favorites.ID || updated.Russian != "trip" {
		t.Fatalf("updated shared word = %#v", updated)
	}
	fromOriginal, err := words.GetByID(ctx, travel.ID, first.ID)
	if err != nil || fromOriginal.Russian != "trip" {
		t.Fatalf("shared edit not visible in original topic: %#v, err=%v", fromOriginal, err)
	}

	if err := topics.Delete(ctx, travel.ID); err != nil {
		t.Fatal(err)
	}
	remaining, err := words.GetByID(ctx, favorites.ID, first.ID)
	if err != nil {
		t.Fatalf("shared word did not survive topic deletion: %v", err)
	}
	if remaining.Notes != "edited from favorites" {
		t.Fatalf("shared word notes = %q", remaining.Notes)
	}

	if err := words.Delete(ctx, favorites.ID, first.ID); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM words WHERE id = ?`, first.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("unlinked word rows = %d, want 0", count)
	}
}

func TestWordTopicsMigrationMergesLegacyDuplicates(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`
		CREATE TABLE topics (id INTEGER PRIMARY KEY, language_track_id INTEGER NOT NULL DEFAULT 1, name TEXT NOT NULL);
		CREATE TABLE words (
			id INTEGER PRIMARY KEY,
			topic_id INTEGER NOT NULL,
			spanish TEXT NOT NULL,
			spanish_key TEXT NOT NULL,
			russian TEXT NOT NULL,
			russian_key TEXT NOT NULL
		);
		CREATE TABLE training_attempts (
			id INTEGER PRIMARY KEY,
			word_id INTEGER NOT NULL,
			direction TEXT NOT NULL,
			question TEXT NOT NULL,
			expected TEXT NOT NULL,
			response TEXT NOT NULL,
			is_correct INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE training_progress (
			word_id INTEGER NOT NULL,
			direction TEXT NOT NULL,
			seen_count INTEGER NOT NULL,
			correct_count INTEGER NOT NULL,
			wrong_count INTEGER NOT NULL,
			correct_streak INTEGER NOT NULL,
			recent_pain INTEGER NOT NULL,
			last_seen_at DATETIME,
			last_correct_at DATETIME,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY(word_id, direction)
		);
		INSERT INTO topics(id, name) VALUES(1, 'Travel'), (2, 'Favorites');
		INSERT INTO words(id, topic_id, spanish, spanish_key, russian, russian_key) VALUES
			(10, 1, 'viaje', 'viaje', 'journey', 'journey'),
			(20, 2, 'Viaje', 'viaje', 'Journey', 'journey');
		INSERT INTO training_attempts(id, word_id, direction, question, expected, response, is_correct)
		VALUES(1, 20, 'spanish_to_russian', 'viaje', 'journey', 'trip', 0);
		INSERT INTO training_progress(
			word_id, direction, seen_count, correct_count, wrong_count,
			correct_streak, recent_pain, updated_at
		) VALUES
			(10, 'spanish_to_russian', 2, 2, 0, 2, 0, CURRENT_TIMESTAMP),
			(20, 'spanish_to_russian', 3, 1, 2, 0, 4, CURRENT_TIMESTAMP);
	`)
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../migrations/008_word_topics.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}

	var words, links, attemptWord, seen, correct, wrong, pain int
	if err := db.QueryRow(`SELECT COUNT(*) FROM words`).Scan(&words); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM word_topics WHERE word_id = 10`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT word_id FROM training_attempts WHERE id = 1`).Scan(&attemptWord); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`
		SELECT seen_count, correct_count, wrong_count, recent_pain
		FROM training_progress WHERE word_id = 10 AND direction = 'spanish_to_russian'
	`).Scan(&seen, &correct, &wrong, &pain); err != nil {
		t.Fatal(err)
	}
	if words != 1 || links != 2 || attemptWord != 10 || seen != 5 || correct != 3 || wrong != 2 || pain != 4 {
		t.Fatalf("migration result words=%d links=%d attemptWord=%d seen=%d correct=%d wrong=%d pain=%d", words, links, attemptWord, seen, correct, wrong, pain)
	}
}
