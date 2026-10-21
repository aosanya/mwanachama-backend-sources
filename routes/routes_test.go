package routes_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	sources "github.com/aosanya/mwanachama-backend-sources"
	"github.com/aosanya/mwanachama-backend-sources/routes"
)

type callerKey struct{}

func callerFrom(ctx context.Context) string {
	id, _ := ctx.Value(callerKey{}).(string)
	return id
}

func newManager(t *testing.T) sources.SourceManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := sources.LoadSpec("../sources.wakala.json")
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	s.Instance = "routestest"
	if err := sources.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	sm, err := sources.NewSourceManager(db, s, sources.WithProfiles(func(_ context.Context, id string) (sources.Profile, error) {
		return sources.Profile{Name: "Name of " + id, Role: "Partner"}, nil
	}))
	if err != nil {
		t.Fatalf("NewSourceManager: %v", err)
	}
	return sm
}

func newServer(t *testing.T, caller string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for _, rt := range routes.RoutesFor(newManager(t), routes.Mount{Caller: callerFrom}) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), callerKey{}, caller)))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, srv *httptest.Server, method, path, body string) (int, map[string]any) {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, _ := http.NewRequest(method, srv.URL+path, rdr)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	var out map[string]any
	raw, _ := io.ReadAll(res.Body)
	_ = json.Unmarshal(raw, &out)
	return res.StatusCode, out
}

func TestRoutesAreTheDeclaredTable(t *testing.T) {
	var got []string
	for _, rt := range routes.Routes(newManager(t)) {
		got = append(got, rt.Pattern(""))
	}
	sort.Strings(got)
	want := []string{
		"DELETE /annotations/{annotationID}",
		"GET /annotations",
		"GET /annotations/{annotationID}",
		"GET /sources",
		"GET /sources/{sourceID}",
		"GET /sources/{sourceID}/chunks",
		"GET /verifications",
		"POST /annotations",
		"POST /chunks/search",
		"POST /sources",
		"POST /sources/{sourceID}/chunks",
		"POST /sources/{sourceID}/documents",
		"POST /verifications",
		"PUT /sources/{sourceID}",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("routes:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestEverySentinelIsMapped(t *testing.T) {
	missing, err := routes.Table.UnmappedSentinels()
	if err != nil {
		t.Fatalf("UnmappedSentinels: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("sentinels with no status in the operations file: %v", missing)
	}
}

func TestEveryRouteIsGated(t *testing.T) {
	if len(routes.AnonymousActions) != 0 {
		t.Fatalf("AnonymousActions = %v — every sources door is the mounting host's to gate", routes.AnonymousActions)
	}
	if n := len(routes.PublicRoutes(newManager(t))); n != 0 {
		t.Fatalf("%d public routes, want none", n)
	}
}

func TestEveryOperationIsATool(t *testing.T) {
	tools, err := routes.BuildTools(newManager(t), routes.Mount{})
	if err != nil {
		t.Fatalf("BuildTools: %v", err)
	}
	if len(tools) != 14 {
		t.Fatalf("got %d tools, want 14", len(tools))
	}
}

func TestTheCallerOutranksTheBodyOnVerify(t *testing.T) {
	srv := newServer(t, "actor-9")
	_, src := do(t, srv, "POST", "/sources", `{"title":"Employment Act, 2007","created_by":"someone-else"}`)
	if src["created_by"] != "actor-9" {
		t.Fatalf("created_by = %v, want the caller", src["created_by"])
	}

	status, v := do(t, srv, "POST", "/verifications",
		`{"subject_role":"source","subject_id":"`+src["id"].(string)+`","status":"verified","verified_by":"actor-1","verifier_role":"Chief Justice"}`)
	if status != http.StatusCreated {
		t.Fatalf("status = %d, body %v", status, v)
	}
	if v["verified_by"] != "actor-9" || v["verifier_role"] != "Partner" || v["verifier_name"] != "Name of actor-9" {
		t.Fatalf("verification = %v, want the caller and their looked-up role", v)
	}
}

func TestVerifyWithNoCallerIsUnauthorized(t *testing.T) {
	srv := newServer(t, "")
	_, src := do(t, srv, "POST", "/sources", `{"title":"Employment Act, 2007"}`)
	status, _ := do(t, srv, "POST", "/verifications",
		`{"subject_role":"source","subject_id":"`+src["id"].(string)+`","status":"verified","verified_by":"actor-1"}`)
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
}

func TestTheAddressOutranksTheBodyOnAChunk(t *testing.T) {
	srv := newServer(t, "actor-9")
	_, a := do(t, srv, "POST", "/sources", `{"title":"A"}`)
	_, b := do(t, srv, "POST", "/sources", `{"title":"B"}`)
	status, c := do(t, srv, "POST", "/sources/"+a["id"].(string)+"/chunks",
		`{"source_id":"`+b["id"].(string)+`","text":"passage"}`)
	if status != http.StatusCreated {
		t.Fatalf("status = %d, body %v", status, c)
	}
	if c["source_id"] != a["id"] {
		t.Fatalf("source_id = %v, want the address's %v", c["source_id"], a["id"])
	}
}

func TestMissingSourceIs404(t *testing.T) {
	srv := newServer(t, "actor-9")
	if status, _ := do(t, srv, "GET", "/sources/missing", ""); status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
}

func TestTheAddressOutranksTheBodyOnADocument(t *testing.T) {
	srv := newServer(t, "actor-9")
	_, a := do(t, srv, "POST", "/sources", `{"title":"A"}`)
	_, b := do(t, srv, "POST", "/sources", `{"title":"B"}`)
	status, _ := do(t, srv, "POST", "/sources/"+a["id"].(string)+"/documents",
		`{"source_id":"`+b["id"].(string)+`","text":"# H\n\npassage","format":"markdown"}`)
	if status != http.StatusCreated {
		t.Fatalf("status = %d", status)
	}
	if n := chunkCount(t, srv, b["id"].(string)); n != 0 {
		t.Fatalf("the body's source got %d chunks", n)
	}
	if n := chunkCount(t, srv, a["id"].(string)); n != 1 {
		t.Fatalf("the address's source got %d chunks, want 1", n)
	}
}

func chunkCount(t *testing.T, srv *httptest.Server, sourceID string) int {
	t.Helper()
	res, err := http.Get(srv.URL + "/sources/" + sourceID + "/chunks")
	if err != nil {
		t.Fatalf("list chunks: %v", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		var wrapped struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.Unmarshal(raw, &wrapped); err != nil {
			t.Fatalf("list chunks: %s", raw)
		}
		items = wrapped.Items
	}
	return len(items)
}
