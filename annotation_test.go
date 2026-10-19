package sources_test

import (
	"context"
	"errors"
	"testing"

	sources "github.com/aosanya/mwanachama-backend-sources"
)

func TestAnnotationNeedsAnExistingSource(t *testing.T) {
	mgr := newTestManager(t)
	_, err := mgr.CreateAnnotation(context.Background(), sources.Annotation{
		SourceID: "missing", TargetModule: "git", TargetID: "blob-1",
	})
	if !errors.Is(err, sources.ErrSourceNotFound) {
		t.Fatalf("err = %v, want ErrSourceNotFound", err)
	}
}

func TestAnnotationNeedsATarget(t *testing.T) {
	mgr := newTestManager(t)
	src := mustSource(t, mgr, "Constitution of Kenya, 2010")
	_, err := mgr.CreateAnnotation(context.Background(), sources.Annotation{SourceID: src.ID, TargetModule: "git"})
	if !errors.Is(err, sources.ErrInvalidAnnotation) {
		t.Fatalf("err = %v, want ErrInvalidAnnotation", err)
	}
}

func TestListAnnotationsOnOneItemOfContent(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	src := mustSource(t, mgr, "Constitution of Kenya, 2010")

	for _, target := range []string{"blob-1", "blob-2", "blob-1"} {
		if _, err := mgr.CreateAnnotation(ctx, sources.Annotation{
			SourceID: src.ID, TargetModule: "git", TargetID: target,
			Anchor: map[string]any{"start": 10, "end": 42},
			Quote:  "Every person is equal before the law",
		}); err != nil {
			t.Fatalf("CreateAnnotation: %v", err)
		}
	}

	got, err := mgr.ListAnnotations(ctx, sources.AnnotationFilter{TargetModule: "git", TargetID: "blob-1"})
	if err != nil {
		t.Fatalf("ListAnnotations: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d annotations, want 2", len(got))
	}
	if got[0].Anchor["end"] != float64(42) {
		t.Fatalf("anchor did not survive the round trip: %+v", got[0].Anchor)
	}
}

func TestDeleteAnnotationKeepsItsVerifications(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	src := mustSource(t, mgr, "Constitution of Kenya, 2010")
	ann, _ := mgr.CreateAnnotation(ctx, sources.Annotation{SourceID: src.ID, TargetModule: "git", TargetID: "blob-1"})
	if _, err := mgr.Verify(ctx, sources.Verification{
		SubjectRole: "annotation", SubjectID: ann.ID, Status: "disputed", VerifiedBy: "actor-2",
	}); err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if err := mgr.DeleteAnnotation(ctx, ann.ID); err != nil {
		t.Fatalf("DeleteAnnotation: %v", err)
	}
	if _, err := mgr.GetAnnotation(ctx, ann.ID); !errors.Is(err, sources.ErrAnnotationNotFound) {
		t.Fatalf("err = %v, want ErrAnnotationNotFound", err)
	}
	if err := mgr.DeleteAnnotation(ctx, ann.ID); !errors.Is(err, sources.ErrAnnotationNotFound) {
		t.Fatalf("second delete err = %v, want ErrAnnotationNotFound", err)
	}
	history, _ := mgr.ListVerifications(ctx, sources.VerificationFilter{SubjectRole: "annotation", SubjectID: ann.ID})
	if len(history) != 1 {
		t.Fatalf("history has %d entries, want the dispute kept", len(history))
	}
}
