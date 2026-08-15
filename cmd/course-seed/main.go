package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"repetidor/internal/coursepack"
	"repetidor/internal/sqlite"
	"repetidor/internal/storage"
)

func main() {
	file := flag.String("file", "", "path to a repetidor-course JSON file")
	database := flag.String("sqlite-path", "./repetidor.sqlite3", "path to the Repetidor SQLite database")
	migrations := flag.String("migrations", "./migrations", "path to database migrations")
	trackID := flag.Int64("track-id", 1, "destination language track ID")
	preview := flag.Bool("preview", false, "validate and summarize without saving")
	flag.Parse()
	if *file == "" {
		log.Fatal("-file is required")
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("read course file: %v", err)
	}
	value, err := coursepack.Decode(raw)
	if err != nil {
		log.Fatalf("validate course file: %v", err)
	}
	db, err := sqlite.Open(*database)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	if err = sqlite.Migrate(db, *migrations); err != nil {
		log.Fatalf("migrate database: %v", err)
	}
	store := sqlite.NewCoursePackageStore(db)
	ctx := context.Background()
	if *preview {
		summary, previewErr := store.Preview(ctx, *trackID, value)
		if previewErr != nil {
			log.Fatal(previewErr)
		}
		printSummary("preview", 0, summary)
		return
	}
	id, summary, err := store.Import(ctx, *trackID, value)
	if errors.Is(err, storage.ErrCoursePackageDuplicate) {
		printSummary("already imported", id, summary)
		return
	}
	if err != nil {
		log.Fatalf("import course: %v", err)
	}
	printSummary("imported", id, summary)
}

func printSummary(status string, id int64, summary storage.CoursePackageSummary) {
	fmt.Printf("course %s", status)
	if id > 0 {
		fmt.Printf(" (id=%d)", id)
	}
	fmt.Printf(": blocks=%d exercises=%d topics=%d words=%d duplicates=%d\n", summary.Blocks, summary.Exercises, summary.Topics, summary.Words, summary.Duplicates)
}
