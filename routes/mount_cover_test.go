package routes_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	"github.com/aosanya/mwanachama-backend-sources/routes"
)

func patterns(rs []routes.Route) []string {
	var out []string
	for _, rt := range rs {
		out = append(out, rt.Pattern(""))
	}
	sort.Strings(out)
	return out
}

func TestBuildReturnsTheSameAddressesAsRoutes(t *testing.T) {
	sm := newManager(t)
	built, err := routes.Build(sm)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if strings.Join(patterns(built), "\n") != strings.Join(patterns(routes.Routes(sm)), "\n") {
		t.Fatalf("Build and Routes disagree on the address table")
	}
	if len(built) != 14 {
		t.Fatalf("Build returned %d routes, want 14", len(built))
	}
}

func TestBuildForCarriesTheMountsCaller(t *testing.T) {
	sm := newManager(t)
	built, err := routes.BuildFor(sm, routes.Mount{Caller: func(context.Context) string { return "actor-3" }})
	if err != nil {
		t.Fatalf("BuildFor: %v", err)
	}
	mux := http.NewServeMux()
	for _, rt := range built {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/sources", strings.NewReader(`{"title":"Act"}`)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var src map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &src)
	if src["created_by"] != "actor-3" {
		t.Fatalf("created_by = %v, want the mount's caller", src["created_by"])
	}
}

func TestSplitPutsEveryRouteOnTheGatedSide(t *testing.T) {
	sm := newManager(t)
	split := routes.Split(sm, routes.Mount{})
	if len(split.Anonymous) != 0 {
		t.Fatalf("anonymous routes: %v", patterns(split.Anonymous))
	}
	if len(split.Gated) != 14 {
		t.Fatalf("gated routes = %d, want 14", len(split.Gated))
	}
	if strings.Join(patterns(routes.GatedRoutes(sm, routes.Mount{})), "\n") != strings.Join(patterns(split.Gated), "\n") {
		t.Fatalf("GatedRoutes differs from Split.Gated")
	}
}

func TestAGatedRouteRefusesWhenTheAuthorizerDenies(t *testing.T) {
	var asked []string
	mount := routes.Mount{
		Caller: func(context.Context) string { return "actor-1" },
		Authorize: func(_ context.Context, action string) error {
			asked = append(asked, action)
			return dispatch.ErrForbidden
		},
	}
	mux := http.NewServeMux()
	for _, rt := range routes.GatedRoutes(newManager(t), mount) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/sources", strings.NewReader(`{"title":"Act"}`)))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if len(asked) != 1 || asked[0] == "" {
		t.Fatalf("authorizer asked %v, want one action", asked)
	}
}

func TestToolsAreNamedAndSchemaBearing(t *testing.T) {
	tools := routes.Tools(newManager(t), routes.Mount{})
	if len(tools) != 14 {
		t.Fatalf("got %d tools, want 14", len(tools))
	}
	seen := map[string]bool{}
	for _, tl := range tools {
		if tl.Name == "" || tl.Action == "" || len(tl.InputSchema) == 0 {
			t.Fatalf("incomplete tool: %+v", tl)
		}
		if seen[tl.Name] {
			t.Fatalf("duplicate tool name %q", tl.Name)
		}
		seen[tl.Name] = true
		if !json.Valid(tl.InputSchema) {
			t.Fatalf("tool %q has an invalid input schema", tl.Name)
		}
	}
}

func TestAToolRefusesWhenTheAuthorizerDenies(t *testing.T) {
	mount := routes.Mount{
		Caller:    func(context.Context) string { return "actor-1" },
		Authorize: func(context.Context, string) error { return dispatch.ErrForbidden },
	}
	tools := routes.Tools(newManager(t), mount)
	for _, tl := range tools {
		if tl.Action == "" {
			continue
		}
		_, err := tl.Invoke(context.Background(), json.RawMessage(`{}`))
		if !errors.Is(err, dispatch.ErrForbidden) {
			t.Fatalf("tool %q err = %v, want forbidden", tl.Name, err)
		}
	}
}

func TestFieldsMarkStoreOwnedAndGeneratedColumnsReadOnly(t *testing.T) {
	fields, err := routes.Fields()
	if err != nil {
		t.Fatalf("Fields: %v", err)
	}
	for _, key := range []string{"Source.created_by", "Annotation.created_by", "Verification.verified_by", "Verification.verifier_name", "Verification.verifier_role", "Source.id"} {
		doc, ok := fields[key]
		if !ok {
			t.Fatalf("no field doc for %s", key)
		}
		if !doc.ReadOnly {
			t.Fatalf("%s should be read-only", key)
		}
	}
}

func TestFieldsCarryTheInputDocumentation(t *testing.T) {
	fields, err := routes.Fields()
	if err != nil {
		t.Fatalf("Fields: %v", err)
	}
	text := fields["Document.text"]
	if !text.Required || text.Description == "" {
		t.Fatalf("Document.text = %+v, want required and described", text)
	}
	format := fields["Document.format"]
	if strings.Join(format.Values, ",") != "text,markdown" {
		t.Fatalf("Document.format values = %v", format.Values)
	}
	for _, key := range []string{"ChunkQuery.embedding", "ChunkQuery.embedding_model"} {
		if !fields[key].Required {
			t.Fatalf("%s should be required", key)
		}
	}
	role := fields["VerificationFilter.subject_role"]
	if strings.Join(role.Values, ",") != "source,annotation" {
		t.Fatalf("subject_role values = %v", role.Values)
	}
}

func sourceCreateTool(t *testing.T, mount routes.Mount) routes.Tool {
	t.Helper()
	for _, tl := range routes.Tools(newManager(t), mount) {
		if strings.HasSuffix(tl.Action, "source.create") {
			return tl
		}
	}
	t.Fatalf("no source create tool")
	return routes.Tool{}
}

func TestACreateToolStampsTheCaller(t *testing.T) {
	create := sourceCreateTool(t, routes.Mount{Caller: func(context.Context) string { return "actor-7" }})
	out, err := create.Invoke(context.Background(), json.RawMessage(`{"title":"Act"}`))
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	raw, _ := json.Marshal(out)
	var got map[string]any
	_ = json.Unmarshal(raw, &got)
	if got["created_by"] != "actor-7" {
		t.Fatalf("created_by = %v, want the caller", got["created_by"])
	}
}

func TestACreateToolRefusesAnArgumentNamingTheCreator(t *testing.T) {
	create := sourceCreateTool(t, routes.Mount{Caller: func(context.Context) string { return "actor-7" }})
	_, err := create.Invoke(context.Background(), json.RawMessage(`{"title":"Act","created_by":"intruder"}`))
	if err == nil || !strings.Contains(err.Error(), "created_by") {
		t.Fatalf("err = %v, want a refusal naming created_by", err)
	}
}
