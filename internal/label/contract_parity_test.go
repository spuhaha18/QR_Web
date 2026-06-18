package label

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// TestFrontendContractParity guards the polyglot domain contract: the Svelte
// frontend re-encodes the same rules (doc types, binder sizes, required-field
// lists) in web/frontend/src/lib/domain.ts. This white-box test (package label)
// parses that file and asserts its constants match the backend value objects,
// so a change on one side that is not mirrored on the other fails CI instead of
// drifting silently. See doc/plan/ddd-refactor.md Phase 7.
func TestFrontendContractParity(t *testing.T) {
	wd, err := os.Getwd() // .../internal/label
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	domainPath := filepath.Join(root, "web", "frontend", "src", "lib", "domain.ts")
	raw, err := os.ReadFile(domainPath)
	if err != nil {
		t.Skipf("frontend domain.ts not found at %s: %v", domainPath, err)
	}
	src := string(raw)

	// DOC_TYPES must equal the backend doc-type codes.
	wantDocTypes := []string{DocTypeEquipment.Code(), DocTypeProject.Code()}
	if got := stringArray(t, src, "DOC_TYPES"); !equalStrings(got, wantDocTypes) {
		t.Errorf("DOC_TYPES: frontend %v, backend %v", got, wantDocTypes)
	}

	// BINDER_SIZES must equal the backend's valid binder set (sorted).
	wantBinders := sortedBinderKeys()
	if got := intArray(t, src, "BINDER_SIZES"); !equalInts(got, wantBinders) {
		t.Errorf("BINDER_SIZES: frontend %v, backend %v", got, wantBinders)
	}

	// Required-field lists must match exactly (same fields, same order).
	if got := stringArray(t, src, "REQUIRED_EQUIPMENT_FIELDS"); !equalStrings(got, EquipmentRequiredFields) {
		t.Errorf("REQUIRED_EQUIPMENT_FIELDS: frontend %v, backend %v", got, EquipmentRequiredFields)
	}
	if got := stringArray(t, src, "REQUIRED_PROJECT_FIELDS"); !equalStrings(got, ProjectRequiredFields) {
		t.Errorf("REQUIRED_PROJECT_FIELDS: frontend %v, backend %v", got, ProjectRequiredFields)
	}
}

// arrayBody extracts the text between the first "NAME ... = [" and the next "]".
func arrayBody(t *testing.T, src, name string) string {
	t.Helper()
	re := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(name) + `\b[^=]*=\s*\[(.*?)\]`)
	m := re.FindStringSubmatch(src)
	if m == nil {
		t.Fatalf("could not find array %s in domain.ts", name)
	}
	return m[1]
}

func stringArray(t *testing.T, src, name string) []string {
	body := arrayBody(t, src, name)
	re := regexp.MustCompile(`'([^']*)'`)
	var out []string
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		out = append(out, m[1])
	}
	return out
}

func intArray(t *testing.T, src, name string) []int {
	body := arrayBody(t, src, name)
	re := regexp.MustCompile(`\d+`)
	var out []int
	for _, m := range re.FindAllString(body, -1) {
		n, _ := strconv.Atoi(m)
		out = append(out, n)
	}
	return out
}

func sortedBinderKeys() []int {
	out := make([]int, 0, len(binderColumnWidth))
	for b := range binderColumnWidth {
		out = append(out, int(b))
	}
	// insertion sort (tiny set) to avoid importing sort for one use.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
