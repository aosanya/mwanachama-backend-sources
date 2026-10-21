package mcp_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	sources "github.com/aosanya/mwanachama-backend-sources"
	sourcesmcp "github.com/aosanya/mwanachama-backend-sources/mcp"
	"github.com/aosanya/mwanachama-backend-sources/routes"
)

func managerFor(t *testing.T, instance string) sources.SourceManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	s, err := sources.SpecFor(instance)
	if err != nil {
		t.Fatal(err)
	}
	if err := sources.Provision(db, s); err != nil {
		t.Fatal(err)
	}
	m, err := sources.NewSourceManager(db, s)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func resolvedSession(t *testing.T, held map[string]sources.SourceManager) *mcp.ClientSession {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "t", Version: "1"}, nil)
	sourcesmcp.RegisterResolvedTools(server, func(_ context.Context, id string) (sources.SourceManager, error) {
		if m, ok := held[id]; ok {
			return m, nil
		}
		return nil, errors.New("not found")
	}, routes.Mount{Caller: func(context.Context) string { return "actor-1" }})
	ct, st := mcp.NewInMemoryTransports()
	if _, err := server.Connect(context.Background(), st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "1"}, nil).Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func toolNamed(t *testing.T, cs *mcp.ClientSession, suffix string) string {
	t.Helper()
	list, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range list.Tools {
		if len(tool.Name) >= len(suffix) && tool.Name[len(tool.Name)-len(suffix):] == suffix {
			return tool.Name
		}
	}
	t.Fatalf("no tool ending %q", suffix)
	return ""
}

func TestAResolvedToolReachesTheNamedInstanceOnly(t *testing.T) {
	a, b := managerFor(t, "alpha"), managerFor(t, "beta")
	if _, err := a.CreateSource(context.Background(), sources.Source{Title: "Held by alpha", Kind: "statute"}); err != nil {
		t.Fatal(err)
	}
	cs := resolvedSession(t, map[string]sources.SourceManager{"alpha": a, "beta": b})
	list := toolNamed(t, cs, "source_list")
	for instance, want := range map[string]int{"alpha": 1, "beta": 0} {
		res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: list, Arguments: map[string]any{"instance_id": instance}})
		if err != nil || res.IsError {
			t.Fatalf("%s: %v %+v", instance, err, res)
		}
		items, _ := res.StructuredContent.(map[string]any)["items"].([]any)
		if len(items) != want {
			t.Fatalf("%s: %d sources, want %d", instance, len(items), want)
		}
	}
}

func TestAResolvedToolRefusesAnUnknownOrMissingInstance(t *testing.T) {
	cs := resolvedSession(t, map[string]sources.SourceManager{"alpha": managerFor(t, "alpha")})
	list := toolNamed(t, cs, "source_list")
	for _, args := range []map[string]any{{"instance_id": "nobody"}, {}} {
		res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: list, Arguments: args})
		if err == nil && !res.IsError {
			t.Fatalf("%v: answered", args)
		}
	}
}
