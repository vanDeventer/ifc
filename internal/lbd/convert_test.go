package lbd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/vanDeventer/ifc/internal/cottage"
	"github.com/vanDeventer/ifc/internal/step"
)

func convert(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	if err := cottage.Build(cottage.Default(), time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)).
		Write(&buf); err != nil {
		t.Fatal(err)
	}
	parsed, err := step.Parse(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	ttl, err := Convert(parsed, Options{Base: "https://example.org/c/", Source: "cottage.ifc"})
	if err != nil {
		t.Fatal(err)
	}
	return ttl
}

// The point of the conversion is that these edges survive it.
func TestCarriesTheRelationships(t *testing.T) {
	ttl := convert(t)
	for _, want := range []string{
		"a bot:Site",
		"bot:hasBuilding",
		"bot:hasStorey",
		"bot:hasSpace",
		"bot:containsElement",
		"bot:adjacentElement", // space boundaries
		"bot:hostsElement",    // a wall hosting a window
		"a bot:Interface",     // the reified boundary, carrying inside/outside
		"bpo:realisesObject",  // occurrence to product
		"a fso:DistributionSystem",
		"fso:hasComponent",
		"a cot:SoftwareSystem", // the mbaigo systems
		"a skos:ConceptScheme",
		"dcterms:subject",
		"fog:asIfc_v2x4", // where the geometry stayed behind
	} {
		if !strings.Contains(ttl, want) {
			t.Errorf("conversion lost %q", want)
		}
	}
}

// Classes are only emitted from the checked table, so a class that does not
// exist upstream cannot appear.
func TestOnlyKnownClasses(t *testing.T) {
	ttl := convert(t)
	for _, want := range []string{
		"beo:Wall", "beo:Wall-PARTITIONING", "beo:Window", "beo:Door", "beo:Slab-FLOOR",
		"beo:Roof-HIP_ROOF", "mep:SpaceHeater-CONVECTOR", "mep:Sensor-WINDSENSOR",
		"mep:SanitaryTerminal-TOILETPAN", "mep:PipeSegment", "fso:Segment", "fso:Terminal",
	} {
		if !strings.Contains(ttl, want) {
			t.Errorf("expected class %q", want)
		}
	}
	// IfcFurnishingElement has no building element class; it must not be given one.
	if strings.Contains(ttl, "beo:FurnishingElement") {
		t.Error("emitted beo:FurnishingElement, which BEO does not define")
	}
	if !strings.Contains(ttl, "# No building or distribution element class for:") {
		t.Error("unmapped entities should be reported in a comment")
	}
}

// A Turtle prefixed name cannot contain $, which IFC's identifier alphabet uses.
func TestIdentifiersAreEscaped(t *testing.T) {
	ttl := convert(t)
	for _, line := range strings.Split(ttl, "\n") {
		if i := strings.Index(line, "inst:"); i >= 0 {
			rest := line[i+len("inst:"):]
			if end := strings.IndexAny(rest, " ,;."); end >= 0 {
				rest = rest[:end]
			}
			if strings.Contains(rest, "$") {
				t.Fatalf("unescaped $ in a prefixed name: %s", line)
			}
		}
	}
}
