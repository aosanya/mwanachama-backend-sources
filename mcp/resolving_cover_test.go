package mcp_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	sources "github.com/aosanya/mwanachama-backend-sources"
)

func callTool(t *testing.T, cs *mcp.ClientSession, suffix string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	args["instance_id"] = "alpha"
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: toolNamed(t, cs, suffix), Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", suffix, err)
	}
	return res
}

func succeeded(t *testing.T, cs *mcp.ClientSession, suffix string, args map[string]any) map[string]any {
	t.Helper()
	res := callTool(t, cs, suffix, args)
	if res.IsError {
		t.Fatalf("%s refused: %+v", suffix, res.Content)
	}
	out, _ := res.StructuredContent.(map[string]any)
	return out
}

func alphaSession(t *testing.T) *mcp.ClientSession {
	t.Helper()
	return resolvedSession(t, map[string]sources.SourceManager{"alpha": managerFor(t, "alpha")})
}

func TestASourceIsCreatedReadUpdatedListedByKindThroughTools(t *testing.T) {
	cs := alphaSession(t)
	created := succeeded(t, cs, "source_create", map[string]any{"title": "Land Act", "kind": "statute"})
	id, _ := created["id"].(string)
	if id == "" || created["title"] != "Land Act" {
		t.Fatalf("created %v", created)
	}
	succeeded(t, cs, "source_create", map[string]any{"title": "Daily paper", "kind": "article"})
	if got := succeeded(t, cs, "source_read", map[string]any{"sourceID": id}); got["kind"] != "statute" {
		t.Fatalf("read %v", got)
	}
	updated := succeeded(t, cs, "source_update", map[string]any{"sourceID": id, "title": "Land Act 2012", "kind": "statute"})
	if updated["title"] != "Land Act 2012" {
		t.Fatalf("updated %v", updated)
	}
	statutes, _ := succeeded(t, cs, "source_list", map[string]any{"kind": "statute"})["items"].([]any)
	if len(statutes) != 1 {
		t.Fatalf("kind filter returned %d", len(statutes))
	}
}

func TestReadingAMissingSourceIsRefused(t *testing.T) {
	cs := alphaSession(t)
	if res := callTool(t, cs, "source_read", map[string]any{"sourceID": "nope"}); !res.IsError {
		t.Fatal("a missing source was answered")
	}
}

func TestVerificationsAreAppendedAndListedPerSubject(t *testing.T) {
	cs := alphaSession(t)
	id, _ := succeeded(t, cs, "source_create", map[string]any{"title": "Land Act", "kind": "statute"})["id"].(string)
	other, _ := succeeded(t, cs, "source_create", map[string]any{"title": "Other", "kind": "statute"})["id"].(string)
	for _, status := range []string{"verified", "disputed"} {
		succeeded(t, cs, "verification_create", map[string]any{"subject_id": id, "subject_role": "source", "status": status})
	}
	succeeded(t, cs, "verification_create", map[string]any{"subject_id": other, "subject_role": "source", "status": "verified"})
	items, _ := succeeded(t, cs, "verification_list", map[string]any{"subject_id": id, "subject_role": "source"})["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("history of one source holds %d records, want 2", len(items))
	}
	bad := callTool(t, cs, "verification_create", map[string]any{"subject_id": "ghost", "subject_role": "source", "status": "verified"})
	if !bad.IsError {
		t.Fatal("a verification of a source that does not exist was accepted")
	}
}

func TestAnnotationsAreCreatedFilteredReadAndDeleted(t *testing.T) {
	cs := alphaSession(t)
	id, _ := succeeded(t, cs, "source_create", map[string]any{"title": "Land Act", "kind": "statute"})["id"].(string)
	made := succeeded(t, cs, "annotation_create", map[string]any{"source_id": id, "target_module": "catalog", "target_id": "entry-1", "quote": "section 4", "note": "supports"})
	annotationID, _ := made["id"].(string)
	succeeded(t, cs, "annotation_create", map[string]any{"source_id": id, "target_module": "forms", "target_id": "form-9", "quote": "section 9", "note": "qualifies"})
	if got := succeeded(t, cs, "annotation_read", map[string]any{"annotationID": annotationID}); got["quote"] != "section 4" {
		t.Fatalf("read %v", got)
	}
	items, _ := succeeded(t, cs, "annotation_list", map[string]any{"target_module": "catalog"})["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("module filter returned %d", len(items))
	}
	succeeded(t, cs, "annotation_delete", map[string]any{"annotationID": annotationID})
	if res := callTool(t, cs, "annotation_read", map[string]any{"annotationID": annotationID}); !res.IsError {
		t.Fatal("a deleted annotation was still readable")
	}
}

func TestADocumentIsSplitIntoPassagesAndSearchedByTheMatchingModelOnly(t *testing.T) {
	cs := alphaSession(t)
	id, _ := succeeded(t, cs, "source_create", map[string]any{"title": "Land Act", "kind": "statute"})["id"].(string)
	doc := succeeded(t, cs, "document_create", map[string]any{"sourceID": id, "text": "first passage\n\nsecond passage"})
	if chunks, _ := doc["items"].([]any); len(chunks) != 2 {
		t.Fatalf("document made %v", doc)
	}
	listed, _ := succeeded(t, cs, "chunk_list", map[string]any{"sourceID": id})["items"].([]any)
	if len(listed) != 2 {
		t.Fatalf("listed %d passages", len(listed))
	}
	succeeded(t, cs, "chunk_create", map[string]any{"sourceID": id, "text": "vector one", "embedding": []float64{1, 0}, "embedding_model": "m1"})
	succeeded(t, cs, "chunk_create", map[string]any{"sourceID": id, "text": "vector two", "embedding": []float64{0, 1}, "embedding_model": "m1"})
	succeeded(t, cs, "chunk_create", map[string]any{"sourceID": id, "text": "foreign", "embedding": []float64{1, 0}, "embedding_model": "m2"})
	matches, _ := succeeded(t, cs, "chunk_search", map[string]any{"embedding": []float64{1, 0}, "embedding_model": "m1", "limit": 5})["items"].([]any)
	if len(matches) != 2 {
		t.Fatalf("search returned %d matches, want 2 from model m1", len(matches))
	}
	first, _ := matches[0].(map[string]any)
	if !strings.Contains(toJSON(first), "vector one") {
		t.Fatalf("closest match is %v", first)
	}
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
