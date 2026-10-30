package mcp_test

import (
	"context"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	sources "github.com/aosanya/mwanachama-backend-sources"
	sourcesmcp "github.com/aosanya/mwanachama-backend-sources/mcp"
	"github.com/aosanya/mwanachama-backend-sources/routes"
)

type conversationRecorder struct {
	sources.SourceManager
	mu   sync.Mutex
	seen []string
}

func (r *conversationRecorder) ListSources(ctx context.Context, filter sources.SourceFilter) ([]sources.Source, error) {
	r.mu.Lock()
	r.seen = append(r.seen, dispatch.ConversationOf(ctx))
	r.mu.Unlock()
	return r.SourceManager.ListSources(ctx, filter)
}

func connectedTo(t *testing.T, register func(*mcp.Server)) *mcp.ClientSession {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "t", Version: "1"}, nil)
	register(server)
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

func conversationsSeenBy(t *testing.T, resolved bool, meta mcp.Meta) []string {
	t.Helper()
	recorder := &conversationRecorder{SourceManager: managerFor(t, "app")}
	mount := routes.Mount{Caller: func(context.Context) string { return "actor-1" }}
	cs := connectedTo(t, func(server *mcp.Server) {
		if resolved {
			sourcesmcp.RegisterResolvedTools(server, func(context.Context, string) (sources.SourceManager, error) { return recorder, nil }, mount)
			return
		}
		sourcesmcp.RegisterTools(server, recorder, mount)
	})
	arguments := map[string]any{}
	if resolved {
		arguments["instance_id"] = "app"
	}
	if _, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Meta:      meta,
		Name:      toolNamed(t, cs, "source_list"),
		Arguments: arguments,
	}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return recorder.seen
}

func TestAToolCallCarryingMetaConversationIdReachesTheManagerOnItsContext(t *testing.T) {
	for _, resolved := range []bool{false, true} {
		seen := conversationsSeenBy(t, resolved, mcp.Meta{dispatch.ConversationMetaKey: "chat-42"})
		if len(seen) != 1 || seen[0] != "chat-42" {
			t.Errorf("resolved=%v: the manager saw conversations %q, want exactly [chat-42]", resolved, seen)
		}
	}
}

func TestAToolCallCarryingNoMetaReachesTheManagerWithNoConversation(t *testing.T) {
	for _, resolved := range []bool{false, true} {
		seen := conversationsSeenBy(t, resolved, nil)
		if len(seen) != 1 || seen[0] != "" {
			t.Errorf("resolved=%v: the manager saw conversations %q, want exactly an empty one", resolved, seen)
		}
	}
}
