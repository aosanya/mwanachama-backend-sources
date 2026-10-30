package sources

import (
	"sort"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"
)

var markedReferences = map[string]string{
	"sourceID":     "sources.source",
	"annotationID": "sources.annotation",
}

func TestEveryArgumentNamingASourceOrAnAnnotationIsMarkedWithWhatItRefersTo(t *testing.T) {
	s, err := dispatch.Parse(Operations())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	marked := map[string]int{}
	for _, name := range sortedOperationNames(s) {
		for _, a := range s.Operations[name].Args {
			want, isReference := markedReferences[a.As]
			isReference = isReference && !a.Ignored
			if isReference && a.Refers != want {
				t.Errorf("%s: argument %s refers %q, want %q", name, a.As, a.Refers, want)
			}
			if !isReference && a.Refers != "" {
				t.Errorf("%s: argument %s refers %q but is not one of the things an operator works with", name, a.As, a.Refers)
			}
			if isReference {
				marked[a.As]++
			}
		}
	}
	for as := range markedReferences {
		if marked[as] == 0 {
			t.Errorf("no operation declares %s, so nothing proves it is marked", as)
		}
	}
}

func TestReferencesListsTheMarkedArgumentsAndNoCallerOrWholeRequest(t *testing.T) {
	s, err := dispatch.Parse(Operations())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	references := s.References()
	if len(references) != 7 {
		t.Fatalf("References lists %d arguments, want the 7 source and annotation path ids the file declares", len(references))
	}
	for _, r := range references {
		if markedReferences[r.As] != r.Refers {
			t.Errorf("%s.%s is listed as referring %q", r.Operation, r.As, r.Refers)
		}
		if r.From != dispatch.FromPath {
			t.Errorf("%s.%s comes from %q, want the path", r.Operation, r.As, r.From)
		}
	}
}

func sortedOperationNames(s *dispatch.Spec) []string {
	names := make([]string, 0, len(s.Operations))
	for name := range s.Operations {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
