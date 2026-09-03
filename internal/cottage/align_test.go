package cottage

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// The alignment is only useful if it names rooms the IFC actually contains, so
// check the identifiers against the model rather than trusting they match.
func TestAlignmentNamesRealSpaces(t *testing.T) {
	p := Default()
	var buf bytes.Buffer
	if err := Build(p, time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)).Write(&buf); err != nil {
		t.Fatal(err)
	}
	model := buf.String()
	align := Alignment(p, AlignOptions{Base: "https://example.org/c/"})

	var placed int
	for _, s := range p.Spaces {
		for _, fl := range s.Functional {
			placed++
			want := "alc:" + fl + " cot:locatedIn inst:" + escapeLocal(SpaceGUID(s.Name)) + " ."
			if !strings.Contains(align, want) {
				t.Errorf("alignment is missing %q", want)
			}
		}
		// The identifier the alignment points at has to be in the IFC.
		if !strings.Contains(model, "'"+SpaceGUID(s.Name)+"'") {
			t.Errorf("space %q has id %s, which is not in the model", s.Name, SpaceGUID(s.Name))
		}
	}
	if placed == 0 {
		t.Fatal("no functional locations were placed")
	}
}

// Identity is the wrong relation here and the file must not assert it: two
// functional locations share the open living space, and owl:sameAs is symmetric
// and transitive, so asserting it twice would merge them.
func TestAlignmentDoesNotAssertIdentity(t *testing.T) {
	p := Default()
	align := Alignment(p, AlignOptions{Base: "https://example.org/c/"})
	for _, line := range strings.Split(align, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue // the comment explaining why not is expected to say it
		}
		if strings.Contains(line, "owl:sameAs") {
			t.Errorf("alignment asserts identity: %s", line)
		}
	}

	// And the case that makes it wrong is really present.
	var shared int
	for _, s := range p.Spaces {
		if len(s.Functional) > 1 {
			shared++
		}
	}
	if shared == 0 {
		t.Skip("no room holds more than one functional location any more")
	}
}

// A prefixed name cannot hold the $ that IFC identifiers may contain.
func TestAlignmentEscapesIdentifiers(t *testing.T) {
	align := Alignment(Default(), AlignOptions{Base: "https://example.org/c/"})
	for _, line := range strings.Split(align, "\n") {
		if strings.Contains(line, "inst:") && strings.Contains(line, "$") {
			t.Errorf("unescaped $ in a prefixed name: %s", line)
		}
	}
}
