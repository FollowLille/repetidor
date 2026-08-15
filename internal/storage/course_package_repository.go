package storage

import (
	"context"
	"errors"

	"repetidor/internal/coursepack"
)

var ErrCoursePackageDuplicate = errors.New("this course package was already imported")

type CoursePackageSummary struct {
	Blocks, Exercises, Topics, Words, Duplicates int
	DuplicateCourseID                            int64
}

type CoursePackageRepository interface {
	Preview(ctx context.Context, trackID int64, value coursepack.Package) (CoursePackageSummary, error)
	Import(ctx context.Context, trackID int64, value coursepack.Package) (int64, CoursePackageSummary, error)
	Export(ctx context.Context, courseID int64) (coursepack.Package, error)
}
