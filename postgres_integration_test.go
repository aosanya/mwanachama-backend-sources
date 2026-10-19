//go:build integration

package sources_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	sources "github.com/aosanya/mwanachama-backend-sources"
)

func pgSpec(t *testing.T, path string) (*gorm.DB, *spec.Spec) {
	t.Helper()
	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		t.Skip("POSTGRES_URL is unset")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := sources.LoadSpec(path)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	s.Instance = fmt.Sprintf("pg%06x", rand.Intn(1<<24))

	t.Cleanup(func() {
		for _, o := range s.Objects {
			db.Exec("drop table if exists " + s.TableFor(o))
		}
		db.Exec("drop table if exists " + s.NameRegistryTable())
	})
	return db, s
}

func TestPostgresPropertiesFilterByPath(t *testing.T) {
	db, s := pgSpec(t, "spec/examples/legal.sources.json")
	if err := sources.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := sources.NewSourceManager(db, s)
	if err != nil {
		t.Fatalf("NewSourceManager: %v", err)
	}
	ctx := context.Background()
	for _, j := range []string{"KE", "UG", "KE"} {
		if _, err := mgr.CreateSource(ctx, sources.Source{
			Title: "authority in " + j, Properties: map[string]any{"jurisdiction": j, "rights": "public domain"},
		}); err != nil {
			t.Fatalf("CreateSource: %v", err)
		}
	}
	o, _ := s.ByRole(sources.RoleSource)
	var n int64
	if err := db.Table(s.TableFor(o)).Where("properties #>> '{jurisdiction}' = ?", "KE").Count(&n).Error; err != nil {
		t.Fatalf("count by path: %v", err)
	}
	if n != 2 {
		t.Fatalf("got %d, want 2 Kenyan authorities", n)
	}
}

func TestPostgresSearchRoundTrip(t *testing.T) {
	db, s := pgSpec(t, "sources.wakala.json")
	if err := sources.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := sources.NewSourceManager(db, s)
	if err != nil {
		t.Fatalf("NewSourceManager: %v", err)
	}
	ctx := context.Background()
	src, _ := mgr.CreateSource(ctx, sources.Source{Title: "T"})
	if _, err := mgr.AddChunk(ctx, sources.Chunk{SourceID: src.ID, Text: "x", Embedding: []float64{0.25, 0.75}, EmbeddingModel: "m1"}); err != nil {
		t.Fatalf("AddChunk: %v", err)
	}
	got, err := mgr.SearchChunks(ctx, sources.ChunkQuery{Embedding: []float64{0.25, 0.75}, EmbeddingModel: "m1"})
	if err != nil {
		t.Fatalf("SearchChunks: %v", err)
	}
	if len(got) != 1 || got[0].Chunk.Embedding[1] != 0.75 {
		t.Fatalf("got %+v", got)
	}
}

func TestPostgresEveryDeclaredNameFitsTheIdentifierLimit(t *testing.T) {
	for _, path := range []string{"sources.wakala.json", "spec/examples/legal.sources.json", "spec/examples/newsroom.sources.json"} {
		_, s := pgSpec(t, path)
		for _, o := range s.Objects {
			if n := len(s.TableFor(o)); n > spec.MaxIdentifier {
				t.Errorf("%s needs %d bytes, and postgres truncates at %d", s.RawNameFor(o), n, spec.MaxIdentifier)
			}
			for _, idx := range o.Indexes {
				if n := len(s.IndexFor(o, idx)); n > spec.MaxIdentifier {
					t.Errorf("index %s.%s needs %d bytes, and postgres truncates at %d", o.Name, idx.Name, n, spec.MaxIdentifier)
				}
			}
		}
	}
}
