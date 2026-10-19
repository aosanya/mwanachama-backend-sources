package sources_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	sources "github.com/aosanya/mwanachama-backend-sources"
)

var profiles = map[string]sources.Profile{
	"actor-1": {Name: "Wanjiru Kamau", Role: "Senior Partner"},
	"actor-2": {Name: "Otieno Ouma", Role: "Paralegal"},
}

func lookup(_ context.Context, actorID string) (sources.Profile, error) {
	return profiles[actorID], nil
}

func testSpec(t *testing.T, instance string) *spec.Spec {
	t.Helper()
	s, err := sources.LoadSpec("sources.wakala.json")
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	s.Instance = instance
	return s
}

func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return db
}

func newTestManager(t *testing.T) sources.SourceManager {
	t.Helper()
	db := openDB(t)
	s := testSpec(t, "test")
	if err := sources.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	mgr, err := sources.NewSourceManager(db, s, sources.WithProfiles(lookup))
	if err != nil {
		t.Fatalf("NewSourceManager: %v", err)
	}
	return mgr
}

func mustSource(t *testing.T, mgr sources.SourceManager, title string) sources.Source {
	t.Helper()
	src, err := mgr.CreateSource(context.Background(), sources.Source{Title: title})
	if err != nil {
		t.Fatalf("CreateSource: %v", err)
	}
	return src
}

func TestNewSourceManagerRefusesANilDB(t *testing.T) {
	if _, err := sources.NewSourceManager(nil, testSpec(t, "test")); err == nil {
		t.Fatal("expected an error for a nil db")
	}
}
