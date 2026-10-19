package sources_test

import (
	"path/filepath"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	sources "github.com/aosanya/mwanachama-backend-sources"
)

var allRoles = []string{sources.RoleSource, sources.RoleAnnotation, sources.RoleVerification, sources.RoleChunk}

func shippedSpecs(t *testing.T) []string {
	t.Helper()
	examples, err := filepath.Glob("spec/examples/*.json")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	return append([]string{"sources.wakala.json"}, examples...)
}

func TestEveryExampleFitsTheTypes(t *testing.T) {
	for _, path := range shippedSpecs(t) {
		t.Run(path, func(t *testing.T) {
			s, err := sources.LoadSpec(path)
			if err != nil {
				t.Fatalf("LoadSpec(%s): %v", path, err)
			}
			db := openDB(t)
			if err := sources.Provision(db, s); err != nil {
				t.Fatalf("Provision(%s): %v", path, err)
			}
			if _, err := sources.NewSourceManager(db, s); err != nil {
				t.Fatalf("NewSourceManager(%s): %v", path, err)
			}
		})
	}
}

func TestTwoDomainsCoexist(t *testing.T) {
	db := openDB(t)
	tables := map[string]bool{}
	for _, path := range shippedSpecs(t) {
		s, err := sources.LoadSpec(path)
		if err != nil {
			t.Fatalf("LoadSpec(%s): %v", path, err)
		}
		if err := sources.Provision(db, s); err != nil {
			t.Fatalf("Provision(%s): %v", path, err)
		}
		for _, o := range s.Objects {
			name := s.TableFor(o)
			if tables[name] {
				t.Fatalf("%s: %s collides with a table another domain already declared", path, s.RawNameFor(o))
			}
			tables[name] = true
		}
	}
	if len(tables) < 2*len(allRoles) {
		t.Fatalf("expected at least two domains of %d objects each, got %d tables", len(allRoles), len(tables))
	}
}

func TestRequiredFieldsHaveNoDefault(t *testing.T) {
	b, err := sources.Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}
	for _, o := range b.Objects {
		for _, f := range o.Fields {
			if f.Required && f.Default != "" {
				t.Errorf("%s.%s is both required and defaulted to %q — the default is exactly what would let an omitted value pass unnoticed",
					o.Role, f.Name, f.Default)
			}
		}
	}
}

func TestEveryRoleArrivesWithItsFields(t *testing.T) {
	s, err := sources.SpecFor("app")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	for _, role := range allRoles {
		o, ok := s.ByRole(role)
		if !ok {
			t.Fatalf("the spec fills no object for the role %q", role)
		}
		if len(o.Fields) == 0 {
			t.Errorf("role %q arrived with no fields — a domain spec loaded through spec.Load rather than the blueprint does this", role)
		}
	}
}

func TestTableNamesAreDerivedNotLiteral(t *testing.T) {
	s, err := sources.LoadSpec("sources.wakala.json")
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	o, ok := s.ByRole(sources.RoleSource)
	if !ok {
		t.Fatal("no object fills the source role")
	}
	if got, want := s.RawNameFor(o), "sources_"+spec.DefaultMount+"_sources"; got != want {
		t.Fatalf("RawNameFor = %q, want %q", got, want)
	}
}
