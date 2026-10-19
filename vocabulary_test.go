package sources_test

import (
	"sort"
	"testing"

	sources "github.com/aosanya/mwanachama-backend-sources"
	"github.com/aosanya/mwanachama-backend-sources/models"
)

func declaredValues(t *testing.T, role, field string) []string {
	t.Helper()
	b, err := sources.Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}
	o, ok := b.Object(role)
	if !ok {
		t.Fatalf("the blueprint declares no %q", role)
	}
	for _, f := range o.Fields {
		if f.Name == field {
			out := append([]string(nil), f.Values...)
			sort.Strings(out)
			return out
		}
	}
	t.Fatalf("%s declares no %s", role, field)
	return nil
}

func sameSet(t *testing.T, label string, declared, inGo []string) {
	t.Helper()
	sort.Strings(inGo)
	if len(declared) != len(inGo) {
		t.Fatalf("%s: blueprint declares %v, models compares against %v", label, declared, inGo)
	}
	for i := range inGo {
		if declared[i] != inGo[i] {
			t.Errorf("%s: blueprint has %q where models has %q", label, declared[i], inGo[i])
		}
	}
}

func TestVocabularyMatchesTheBlueprint(t *testing.T) {
	sameSet(t, "verification.subject_role",
		declaredValues(t, sources.RoleVerification, "subject_role"),
		[]string{models.SubjectSource, models.SubjectAnnotation})
	sameSet(t, "verification.status",
		declaredValues(t, sources.RoleVerification, "status"),
		[]string{models.StatusVerified, models.StatusDisputed, models.StatusWithdrawn})
}

func TestEverySubjectRoleIsARoleTheModuleDeclares(t *testing.T) {
	for _, v := range declaredValues(t, sources.RoleVerification, "subject_role") {
		if v != sources.RoleSource && v != sources.RoleAnnotation {
			t.Errorf("subject_role %q names no verifiable role", v)
		}
	}
}
