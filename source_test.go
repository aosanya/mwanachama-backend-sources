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
