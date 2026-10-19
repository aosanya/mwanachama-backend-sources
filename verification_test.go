package sources_test

import (
	"context"
	"errors"
	"testing"

	sources "github.com/aosanya/mwanachama-backend-sources"
)

func TestVerifyRecordsTheVerifiersNameAndRoleAsTheyStand(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	src := mustSource(t, mgr, "Employment Act, 2007")

	v, err := mgr.Verify(ctx, sources.Verification{
		SubjectRole: "source", SubjectID: src.ID, Status: "verified", VerifiedBy: "actor-1",
		VerifierName: "Someone Else", VerifierRole: "Chief Justice",
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if v.VerifierName != "Wanjiru Kamau" || v.VerifierRole != "Senior Partner" {
		t.Fatalf("verifier = %q / %q, want the profile lookup's answer, not the body's", v.VerifierName, v.VerifierRole)
	}

	profiles["actor-1"] = sources.Profile{Name: "Wanjiru Kamau", Role: "Managing Partner"}
	t.Cleanup(func() { profiles["actor-1"] = sources.Profile{Name: "Wanjiru Kamau", Role: "Senior Partner"} })

	history, _ := mgr.ListVerifications(ctx, sources.VerificationFilter{SubjectRole: "source", SubjectID: src.ID})
	if history[0].VerifierRole != "Senior Partner" {
		t.Fatalf("role = %q, want the role held when they vouched", history[0].VerifierRole)
	}
}

func TestVerifyWithoutACallerIsRefused(t *testing.T) {
	mgr := newTestManager(t)
	src := mustSource(t, mgr, "Employment Act, 2007")
	_, err := mgr.Verify(context.Background(), sources.Verification{
		SubjectRole: "source", SubjectID: src.ID, Status: "verified",
	})
	if !errors.Is(err, sources.ErrUnverifiedCaller) {
		t.Fatalf("err = %v, want ErrUnverifiedCaller", err)
	}
}

func TestVerifyRefusesAStatusOutsideTheVocabulary(t *testing.T) {
	mgr := newTestManager(t)
	src := mustSource(t, mgr, "Employment Act, 2007")
	_, err := mgr.Verify(context.Background(), sources.Verification{
		SubjectRole: "source", SubjectID: src.ID, Status: "probably", VerifiedBy: "actor-1",
	})
	if !errors.Is(err, sources.ErrInvalidVerification) {
		t.Fatalf("err = %v, want ErrInvalidVerification", err)
	}
}

func TestVerifyRefusesAMissingStatus(t *testing.T) {
	mgr := newTestManager(t)
	src := mustSource(t, mgr, "Employment Act, 2007")
	_, err := mgr.Verify(context.Background(), sources.Verification{
		SubjectRole: "source", SubjectID: src.ID, VerifiedBy: "actor-1",
	})
	if !errors.Is(err, sources.ErrInvalidVerification) {
		t.Fatalf("err = %v, want ErrInvalidVerification", err)
	}
}

func TestVerifyNeedsAnExistingSubject(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	_, err := mgr.Verify(ctx, sources.Verification{
		SubjectRole: "source", SubjectID: "missing", Status: "verified", VerifiedBy: "actor-1",
	})
	if !errors.Is(err, sources.ErrSourceNotFound) {
		t.Fatalf("source err = %v, want ErrSourceNotFound", err)
	}
	_, err = mgr.Verify(ctx, sources.Verification{
		SubjectRole: "annotation", SubjectID: "missing", Status: "verified", VerifiedBy: "actor-1",
	})
	if !errors.Is(err, sources.ErrAnnotationNotFound) {
		t.Fatalf("annotation err = %v, want ErrAnnotationNotFound", err)
	}
}

func TestTheLatestVerificationComesFirst(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	src := mustSource(t, mgr, "Employment Act, 2007")

	for _, step := range []struct{ by, status string }{
		{"actor-2", "verified"},
		{"actor-1", "disputed"},
	} {
		if _, err := mgr.Verify(ctx, sources.Verification{
			SubjectRole: "source", SubjectID: src.ID, Status: step.status, VerifiedBy: step.by,
		}); err != nil {
			t.Fatalf("Verify: %v", err)
		}
	}

	history, err := mgr.ListVerifications(ctx, sources.VerificationFilter{SubjectRole: "source", SubjectID: src.ID})
	if err != nil {
		t.Fatalf("ListVerifications: %v", err)
	}
	if len(history) != 2 || history[0].Status != "disputed" || history[0].VerifierRole != "Senior Partner" {
		t.Fatalf("history = %+v, want the partner's dispute first", history)
	}
}

func TestVerifyWithoutAProfileLookupRecordsOnlyTheActor(t *testing.T) {
	db := openDB(t)
	s := testSpec(t, "noprofiles")
	if err := sources.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := sources.NewSourceManager(db, s)
	if err != nil {
		t.Fatalf("NewSourceManager: %v", err)
	}
	src := mustSource(t, mgr, "Employment Act, 2007")
	v, err := mgr.Verify(context.Background(), sources.Verification{
		SubjectRole: "source", SubjectID: src.ID, Status: "verified", VerifiedBy: "actor-1",
		VerifierName: "Claimed Name", VerifierRole: "Claimed Role",
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if v.VerifiedBy != "actor-1" || v.VerifierName != "" || v.VerifierRole != "" {
		t.Fatalf("got %+v, want only the actor and no claimed name or role", v)
	}
}
