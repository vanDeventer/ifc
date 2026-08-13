package cottage

import (
	"bytes"
	"math"
	"regexp"
	"strings"
	"testing"
	"time"
)

func shoelace(poly [][2]float64) float64 {
	var s float64
	for i := range poly {
		a, b := poly[i], poly[(i+1)%len(poly)]
		s += a[0]*b[1] - b[0]*a[1]
	}
	return math.Abs(s) / 2
}

// The four rooms along the back wall were given as clear widths, and they have
// to add up to the inside length of the base.
func TestRoomsFillTheBackWall(t *testing.T) {
	p := Default()
	const bath, hall, bed, kitchen = 1900.0, 1000.0, 2690.0, 2350.0
	if got := bath + hall + bed + kitchen; got != p.BaseLength {
		t.Errorf("rooms along the back wall sum to %.0f, base is %.0f", got, p.BaseLength)
	}
}

// The two legs of the L have to close on themselves.
func TestFootprintCloses(t *testing.T) {
	p := Default()
	if got := p.BaseLength - p.RiseWidth; got <= 0 {
		t.Fatalf("rise is wider than the base")
	}
	inside := [][2]float64{
		{0, p.RiseLength - p.BaseWidth},
		{p.BaseLength - p.RiseWidth, p.RiseLength - p.BaseWidth},
		{p.BaseLength - p.RiseWidth, 0},
		{p.BaseLength, 0},
		{p.BaseLength, p.RiseLength},
		{0, p.RiseLength},
	}
	got := shoelace(inside) / 1e6
	want := (p.BaseLength*p.BaseWidth + p.RiseWidth*(p.RiseLength-p.BaseWidth)) / 1e6
	if math.Abs(got-want) > 0.01 {
		t.Errorf("inside footprint %.2f m2, expected %.2f m2", got, want)
	}
	if math.Abs(got-50.10) > 0.01 {
		t.Errorf("inside footprint %.2f m2, expected 50.10 m2", got)
	}
}

// The rooms should account for nearly all of the footprint; the shortfall is
// only the partitions standing inside it.
func TestSpacesCoverTheFootprint(t *testing.T) {
	p := Default()
	var total float64
	for _, s := range p.Spaces {
		total += shoelace(s.Polygon)
	}
	total /= 1e6
	if total < 48.0 || total > 50.11 {
		t.Errorf("rooms total %.2f m2, expected just under the 50.10 m2 footprint", total)
	}
}

// Build panics if an opening falls outside its wall, so this covers placement
// as well as the file coming out well formed.
func TestBuild(t *testing.T) {
	p := Default()
	f := Build(p, time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC))

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	for _, want := range []string{"FILE_SCHEMA(('IFC4'));", "IFCPROJECT(", "IFCBUILDINGSTOREY(", "END-ISO-10303-21;"} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q", want)
		}
	}

	// Every opening must be voided out of a wall and filled by a window or door.
	voids := strings.Count(out, "IFCRELVOIDSELEMENT(")
	fills := strings.Count(out, "IFCRELFILLSELEMENT(")
	if voids != len(p.Openings) || fills != len(p.Openings) {
		t.Errorf("%d openings gave %d voids and %d fills", len(p.Openings), voids, fills)
	}

	// GUIDs are 22 characters and must not collide.
	guid := regexp.MustCompile(`'([0-9A-Za-z_$]{22})',#\d+,`)
	seen := map[string]bool{}
	for _, m := range guid.FindAllStringSubmatch(out, -1) {
		if seen[m[1]] {
			t.Errorf("duplicate GlobalId %s", m[1])
		}
		seen[m[1]] = true
	}
	if len(seen) < 30 {
		t.Errorf("only %d GlobalIds found, expected one per product and relationship", len(seen))
	}
}
