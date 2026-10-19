package sources_test

import (
	"context"
	"errors"
	"testing"

	sources "github.com/aosanya/mwanachama-backend-sources"
)

func addChunk(t *testing.T, mgr sources.SourceManager, sourceID string, pos int, text string, vec []float64, model string) {
	t.Helper()
	if _, err := mgr.AddChunk(context.Background(), sources.Chunk{
		SourceID: sourceID, Position: pos, Text: text, Embedding: vec, EmbeddingModel: model,
	}); err != nil {
		t.Fatalf("AddChunk: %v", err)
	}
}

func TestChunksReadBackInOrder(t *testing.T) {
	mgr := newTestManager(t)
	src := mustSource(t, mgr, "Employment Act, 2007")
	addChunk(t, mgr, src.ID, 2, "third", nil, "")
	addChunk(t, mgr, src.ID, 0, "first", nil, "")
	addChunk(t, mgr, src.ID, 1, "second", nil, "")

	got, err := mgr.ListChunks(context.Background(), src.ID)
	if err != nil {
		t.Fatalf("ListChunks: %v", err)
	}
	if len(got) != 3 || got[0].Text != "first" || got[2].Text != "third" {
		t.Fatalf("got %+v, want first, second, third", got)
	}
}

func TestAChunkNeedsAnExistingSource(t *testing.T) {
	mgr := newTestManager(t)
	_, err := mgr.AddChunk(context.Background(), sources.Chunk{SourceID: "missing", Text: "x"})
	if !errors.Is(err, sources.ErrSourceNotFound) {
		t.Fatalf("err = %v, want ErrSourceNotFound", err)
	}
}

func TestAnEmbeddingNeedsItsModel(t *testing.T) {
	mgr := newTestManager(t)
	src := mustSource(t, mgr, "Employment Act, 2007")
	_, err := mgr.AddChunk(context.Background(), sources.Chunk{SourceID: src.ID, Text: "x", Embedding: []float64{1, 0}})
	if !errors.Is(err, sources.ErrInvalidChunk) {
		t.Fatalf("err = %v, want ErrInvalidChunk", err)
	}
}

func TestANegativePositionIsRefused(t *testing.T) {
	mgr := newTestManager(t)
	src := mustSource(t, mgr, "Employment Act, 2007")
	_, err := mgr.AddChunk(context.Background(), sources.Chunk{SourceID: src.ID, Text: "x", Position: -1})
	if !errors.Is(err, sources.ErrInvalidChunk) {
		t.Fatalf("err = %v, want ErrInvalidChunk", err)
	}
}

func TestSearchRanksByMeaningAndCarriesTheSourceAndItsStanding(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	act := mustSource(t, mgr, "Employment Act, 2007")
	other := mustSource(t, mgr, "Land Act, 2012")

	addChunk(t, mgr, act.ID, 0, "termination notice", []float64{1, 0, 0}, "m1")
	addChunk(t, mgr, act.ID, 1, "annual leave", []float64{0, 1, 0}, "m1")
	addChunk(t, mgr, other.ID, 0, "land registration", []float64{0, 0, 1}, "m1")
	addChunk(t, mgr, other.ID, 1, "another model", []float64{1, 0, 0}, "m2")
	addChunk(t, mgr, other.ID, 2, "wrong width", []float64{1, 0}, "m1")

	if _, err := mgr.Verify(ctx, sources.Verification{
		SubjectRole: "source", SubjectID: act.ID, Status: "verified", VerifiedBy: "actor-1",
	}); err != nil {
		t.Fatalf("Verify: %v", err)
	}

	got, err := mgr.SearchChunks(ctx, sources.ChunkQuery{
		Embedding: []float64{0.9, 0.1, 0}, EmbeddingModel: "m1", Limit: 2,
	})
	if err != nil {
		t.Fatalf("SearchChunks: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d matches, want 2", len(got))
	}
	top := got[0]
	if top.Chunk.Text != "termination notice" || top.Source.Title != "Employment Act, 2007" {
		t.Fatalf("top match = %+v", top)
	}
	if top.Standing == nil || top.Standing.Status != "verified" || top.Standing.VerifierRole != "Senior Partner" {
		t.Fatalf("top match standing = %+v, want the partner's verification", top.Standing)
	}
	for _, m := range got {
		if m.Chunk.EmbeddingModel != "m1" {
			t.Fatalf("a %s chunk was compared with an m1 query", m.Chunk.EmbeddingModel)
		}
	}
}

func TestSearchNarrowsToOneSource(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	act := mustSource(t, mgr, "Employment Act, 2007")
	other := mustSource(t, mgr, "Land Act, 2012")
	addChunk(t, mgr, act.ID, 0, "a", []float64{1, 0}, "m1")
	addChunk(t, mgr, other.ID, 0, "b", []float64{1, 0}, "m1")

	got, err := mgr.SearchChunks(ctx, sources.ChunkQuery{
		Embedding: []float64{1, 0}, EmbeddingModel: "m1", SourceID: other.ID,
	})
	if err != nil {
		t.Fatalf("SearchChunks: %v", err)
	}
	if len(got) != 1 || got[0].Source.ID != other.ID || got[0].Standing != nil {
		t.Fatalf("got %+v, want only the unverified Land Act passage", got)
	}
}

func TestSearchNeedsAVectorAndAModel(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.SearchChunks(ctx, sources.ChunkQuery{EmbeddingModel: "m1"}); !errors.Is(err, sources.ErrInvalidQuery) {
		t.Fatalf("no vector: err = %v, want ErrInvalidQuery", err)
	}
	if _, err := mgr.SearchChunks(ctx, sources.ChunkQuery{Embedding: []float64{1}}); !errors.Is(err, sources.ErrInvalidQuery) {
		t.Fatalf("no model: err = %v, want ErrInvalidQuery", err)
	}
}
