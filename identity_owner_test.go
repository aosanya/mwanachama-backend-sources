package sources_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/identity"

	sources "github.com/aosanya/mwanachama-backend-sources"
)

var _ identity.Owner = sources.SourceManager(nil)

type heldThings struct {
	source       sources.Source
	annotation   sources.Annotation
	verification sources.Verification
	chunk        sources.Chunk
}

func holdOneOfEach(t *testing.T, mgr sources.SourceManager) heldThings {
	t.Helper()
	ctx := context.Background()
	source := mustSource(t, mgr, "Employment Act, 2007")
	annotation, err := mgr.CreateAnnotation(ctx, sources.Annotation{SourceID: source.ID, TargetModule: "git", TargetID: "blob-1"})
	if err != nil {
		t.Fatalf("CreateAnnotation: %v", err)
	}
	verification, err := mgr.Verify(ctx, sources.Verification{SubjectRole: "source", SubjectID: source.ID, Status: "verified", VerifiedBy: "actor-1"})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	chunk, err := mgr.AddChunk(ctx, sources.Chunk{SourceID: source.ID, Text: "A passage."})
	if err != nil {
		t.Fatalf("AddChunk: %v", err)
	}
	return heldThings{source: source, annotation: annotation, verification: verification, chunk: chunk}
}

func (h heldThings) idsByObject() map[string]string {
	return map[string]string{
		"source": h.source.ID, "annotation": h.annotation.ID, "verification": h.verification.ID, "chunk": h.chunk.ID,
	}
}

func TestStableIdentityIsTheRawSourcesInstanceMountObjectAndRowId(t *testing.T) {
	mgr := newTestManager(t)
	held := holdOneOfEach(t, mgr)

	for object, id := range held.idsByObject() {
		got, err := mgr.StableIdentity(context.Background(), object, id)
		if err != nil {
			t.Fatalf("StableIdentity(%s): %v", object, err)
		}
		if want := "sources/test/main/" + object + "/" + id; got != want {
			t.Errorf("StableIdentity(%s) = %q, want %q", object, got, want)
		}
	}
}

func TestResolveGivesBackTheObjectAndIdStableIdentityWasAskedFor(t *testing.T) {
	mgr := newTestManager(t)
	held := holdOneOfEach(t, mgr)

	for object, id := range held.idsByObject() {
		raw, err := mgr.StableIdentity(context.Background(), object, id)
		if err != nil {
			t.Fatalf("StableIdentity(%s): %v", object, err)
		}
		gotObject, gotID, err := mgr.Resolve(context.Background(), raw)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", raw, err)
		}
		if gotObject != object || gotID != id {
			t.Errorf("Resolve(%q) = (%q, %q), want (%q, %q)", raw, gotObject, gotID, object, id)
		}
	}
}

func TestASourcesStableIdentityIsTheSameAfterItIsEdited(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	source := mustSource(t, mgr, "Employment Act, 2007")
	before, err := mgr.StableIdentity(ctx, "source", source.ID)
	if err != nil {
		t.Fatalf("StableIdentity: %v", err)
	}
	source.Title = "Employment Act, 2007 (revised)"
	if _, err := mgr.UpdateSource(ctx, source); err != nil {
		t.Fatalf("UpdateSource: %v", err)
	}
	after, err := mgr.StableIdentity(ctx, "source", source.ID)
	if err != nil {
		t.Fatalf("StableIdentity: %v", err)
	}
	if before != after {
		t.Fatalf("identity moved from %q to %q on an edit", before, after)
	}
}

func TestStableIdentityNamesAMissAndAnObjectItDoesNotKnow(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.StableIdentity(ctx, "source", "missing"); !errors.Is(err, identity.ErrNotFound) {
		t.Errorf("a missing row: err = %v, want identity.ErrNotFound", err)
	}
	if _, err := mgr.StableIdentity(ctx, "invoice", "any"); !errors.Is(err, sources.ErrNoStableIdentity) {
		t.Errorf("an object sources does not hold: err = %v, want ErrNoStableIdentity", err)
	}
}

func TestResolveNamesWhyARawIsNotAnswered(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	source := mustSource(t, mgr, "Employment Act, 2007")
	raw, err := mgr.StableIdentity(ctx, "source", source.ID)
	if err != nil {
		t.Fatalf("StableIdentity: %v", err)
	}
	if err := mgr.DeleteSource(ctx, source.ID); err != nil {
		t.Fatalf("DeleteSource: %v", err)
	}

	cases := map[string]struct {
		raw  string
		want error
	}{
		"a string that does not parse":  {"not a raw", identity.ErrInvalidRaw},
		"another module's raw":          {strings.Replace(raw, "sources/", "agency/", 1), identity.ErrNotOwned},
		"another instance's raw":        {strings.Replace(raw, "/test/", "/other/", 1), identity.ErrNotOwned},
		"another mount's raw":           {strings.Replace(raw, "/main/", "/harvest/", 1), identity.ErrNotOwned},
		"an identity that is not three": {"sources/test/main/" + source.ID, identity.ErrNotFound},
		"an object sources lacks":       {"sources/test/main/invoice/" + source.ID, identity.ErrNotFound},
		"a row that is gone":            {raw, identity.ErrNotFound},
	}
	for name, c := range cases {
		if _, _, err := mgr.Resolve(ctx, c.raw); !errors.Is(err, c.want) {
			t.Errorf("%s: Resolve(%q) err = %v, want %v", name, c.raw, err, c.want)
		}
	}
}

func TestTwoMountsOfOneInstanceDoNotAnswerForEachOthersThings(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()
	managers := map[string]sources.SourceManager{}
	for _, mount := range []string{"harvest", "inputs"} {
		s, err := sources.SpecForMount("test", mount)
		if err != nil {
			t.Fatalf("SpecForMount(%s): %v", mount, err)
		}
		if err := sources.Provision(db, s); err != nil {
			t.Fatalf("Provision(%s): %v", mount, err)
		}
		mgr, err := sources.NewSourceManager(db, s)
		if err != nil {
			t.Fatalf("NewSourceManager(%s): %v", mount, err)
		}
		managers[mount] = mgr
	}
	source := mustSource(t, managers["harvest"], "Employment Act, 2007")
	raw, err := managers["harvest"].StableIdentity(ctx, "source", source.ID)
	if err != nil {
		t.Fatalf("StableIdentity: %v", err)
	}
	if want := "sources/test/harvest/source/" + source.ID; raw != want {
		t.Fatalf("raw = %q, want %q", raw, want)
	}
	if _, _, err := managers["inputs"].Resolve(ctx, raw); !errors.Is(err, identity.ErrNotOwned) {
		t.Fatalf("the other mount: err = %v, want identity.ErrNotOwned", err)
	}
	if _, id, err := managers["harvest"].Resolve(ctx, raw); err != nil || id != source.ID {
		t.Fatalf("its own mount: id = %q err = %v", id, err)
	}
}
