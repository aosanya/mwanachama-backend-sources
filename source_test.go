package sources_test

import (
	"context"
	"errors"
	"testing"

	sources "github.com/aosanya/mwanachama-backend-sources"
)

func TestCreateSourceKeepsItsProperties(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	in := sources.Source{
		Title:    "Employment Act, 2007",
		Kind:     "statute",
		Citation: "No. 11 of 2007",
		Properties: map[string]any{
			"rights":       "public domain",
			"jurisdiction": "KE",
		},
	}
	created, err := mgr.CreateSource(ctx, in)
	if err != nil {
		t.Fatalf("CreateSource: %v", err)
	}
	got, err := mgr.GetSource(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetSource: %v", err)
	}
	if got.Properties["rights"] != "public domain" || got.Properties["jurisdiction"] != "KE" {
		t.Fatalf("properties did not survive the round trip: %+v", got.Properties)
	}
	if got.CreatedAt == "" || got.UpdatedAt != got.CreatedAt {
		t.Fatalf("timestamps not set on create: %+v", got)
	}
}

func TestCreateSourceRefusesAnUntitledSource(t *testing.T) {
	mgr := newTestManager(t)
	_, err := mgr.CreateSource(context.Background(), sources.Source{Kind: "statute"})
	if !errors.Is(err, sources.ErrInvalidSource) {
		t.Fatalf("err = %v, want ErrInvalidSource", err)
	}
}

func TestGetSourceThatDoesNotExist(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetSource(context.Background(), "missing"); !errors.Is(err, sources.ErrSourceNotFound) {
		t.Fatalf("err = %v, want ErrSourceNotFound", err)
	}
}

func TestListSourcesNarrowsByKind(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	for _, in := range []sources.Source{
		{Title: "A", Kind: "statute"},
		{Title: "B", Kind: "judgment"},
		{Title: "C", Kind: "statute"},
	} {
		if _, err := mgr.CreateSource(ctx, in); err != nil {
			t.Fatalf("CreateSource: %v", err)
		}
	}
	got, err := mgr.ListSources(ctx, sources.SourceFilter{Kind: "statute"})
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if len(got) != 2 || got[0].Title != "C" {
		t.Fatalf("got %+v, want C then A", got)
	}
}

func TestUpdateSourceKeepsWhoFiledIt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	created, err := mgr.CreateSource(ctx, sources.Source{Title: "Draft title", CreatedBy: "actor-1"})
	if err != nil {
		t.Fatalf("CreateSource: %v", err)
	}

	updated, err := mgr.UpdateSource(ctx, sources.Source{
		ID: created.ID, Title: "Final title", CreatedBy: "actor-2",
		Properties: map[string]any{"rights": "licensed"},
	})
	if err != nil {
		t.Fatalf("UpdateSource: %v", err)
	}
	got, _ := mgr.GetSource(ctx, created.ID)
	if got.Title != "Final title" || got.Properties["rights"] != "licensed" {
		t.Fatalf("revision lost: %+v", got)
	}
	if got.CreatedBy != "actor-1" || got.CreatedAt != created.CreatedAt {
		t.Fatalf("revision rewrote who filed it or when: %+v", got)
	}
	if updated.UpdatedAt == created.UpdatedAt {
		t.Fatal("updated_at did not move")
	}
}

func TestUpdateSourceCanClearProperties(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	created, _ := mgr.CreateSource(ctx, sources.Source{
		Title: "T", Properties: map[string]any{"rights": "licensed"},
	})
	if _, err := mgr.UpdateSource(ctx, sources.Source{ID: created.ID, Title: "T"}); err != nil {
		t.Fatalf("UpdateSource: %v", err)
	}
	got, _ := mgr.GetSource(ctx, created.ID)
	if len(got.Properties) != 0 {
		t.Fatalf("properties = %v, want them cleared", got.Properties)
	}
}

func TestDeleteSourceRemovesItWithItsChunksAndVerifications(t *testing.T) {
	db := openDB(t)
	s := testSpec(t, "test")
	if err := sources.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := sources.NewSourceManager(db, s, sources.WithProfiles(lookup))
	if err != nil {
		t.Fatalf("NewSourceManager: %v", err)
	}
	ctx := context.Background()
	kept := mustSource(t, mgr, "Kept")
	gone := mustSource(t, mgr, "Gone")
	addChunk(t, mgr, kept.ID, 0, "kept chunk", nil, "")
	addChunk(t, mgr, gone.ID, 0, "gone chunk", nil, "")
	if _, err := mgr.Verify(ctx, sources.Verification{SubjectRole: "source", SubjectID: gone.ID, Status: "verified", VerifiedBy: "actor-1"}); err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if err := mgr.DeleteSource(ctx, gone.ID); err != nil {
		t.Fatalf("DeleteSource: %v", err)
	}

	if _, err := mgr.GetSource(ctx, gone.ID); !errors.Is(err, sources.ErrSourceNotFound) {
		t.Fatalf("GetSource err = %v, want ErrSourceNotFound", err)
	}
	var chunkTable string
	if err := db.Raw("select m.name from sqlite_master m, pragma_table_info(m.name) p where m.type = 'table' and p.name = 'start_offset'").Row().Scan(&chunkTable); err != nil {
		t.Fatalf("chunk table = %q, %v", chunkTable, err)
	}
	var orphaned int64
	if err := db.Table(chunkTable).Where("source_id = ?", gone.ID).Count(&orphaned).Error; err != nil || orphaned != 0 {
		t.Fatalf("chunk rows left behind = %d, %v", orphaned, err)
	}
	if held, err := mgr.ListVerifications(ctx, sources.VerificationFilter{SubjectRole: "source", SubjectID: gone.ID}); err != nil || len(held) != 0 {
		t.Fatalf("verifications of the deleted source = %+v, %v", held, err)
	}
	if _, err := mgr.GetSource(ctx, kept.ID); err != nil {
		t.Fatalf("the other source was lost: %v", err)
	}
	if chunks, err := mgr.ListChunks(ctx, kept.ID); err != nil || len(chunks) != 1 {
		t.Fatalf("chunks of the other source = %+v, %v", chunks, err)
	}
}

func TestDeleteSourceRefusesASourceThatAnnotationsRestOn(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	src := mustSource(t, mgr, "Employment Act, 2007")
	if _, err := mgr.CreateAnnotation(ctx, sources.Annotation{SourceID: src.ID, TargetModule: "git", TargetID: "blob-1"}); err != nil {
		t.Fatalf("CreateAnnotation: %v", err)
	}

	if err := mgr.DeleteSource(ctx, src.ID); !errors.Is(err, sources.ErrSourceInUse) {
		t.Fatalf("err = %v, want ErrSourceInUse", err)
	}
	if _, err := mgr.GetSource(ctx, src.ID); err != nil {
		t.Fatalf("the source was lost: %v", err)
	}
}

func TestDeleteSourceOfAMissingSourceIsNotFound(t *testing.T) {
	mgr := newTestManager(t)
	if err := mgr.DeleteSource(context.Background(), "missing"); !errors.Is(err, sources.ErrSourceNotFound) {
		t.Fatalf("err = %v, want ErrSourceNotFound", err)
	}
}
