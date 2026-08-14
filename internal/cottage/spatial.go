package cottage

import (
	"fmt"
	"math"

	"github.com/vanDeventer/ifc/internal/ifc"
)

// tol is how close two faces must be, in millimetres, to count as touching.
const tol = 1.0

// contain puts each element in the smallest spatial structure that holds it: a
// fixture or device goes in its room, everything else in the storey. IFC allows
// only one container per element, so this is a partition, not an addition.
func (b *builder) contain(p Params, storey ifc.Ref) {
	var loose []ifc.Ref
	inRoom := map[string][]ifc.Ref{}

	for _, r := range b.records {
		if r.roomable {
			for _, s := range p.Spaces {
				if inPolygon(s.Polygon, r.cx, r.cy) {
					r.space = s.Name
					inRoom[s.Name] = append(inRoom[s.Name], r.ref)
					break
				}
			}
		}
		if r.space == "" {
			loose = append(loose, r.ref)
		}
	}

	b.f.Add("IFCRELCONTAINEDINSPATIALSTRUCTURE", ifc.GUID("contained-storey"), b.owner,
		"Ground floor contents", ifc.Null{}, loose, storey)

	for _, s := range p.Spaces {
		if members := inRoom[s.Name]; len(members) > 0 {
			b.f.Add("IFCRELCONTAINEDINSPATIALSTRUCTURE", ifc.GUID("contained-"+s.Name),
				b.owner, s.Name+" contents", ifc.Null{}, members, b.spaces[s.Name])
		}
	}
}

// boundaries records which element bounds which room. This is what turns the
// model into something a graph can be asked questions of: which room is behind
// this wall, which rooms does this door join, what faces outside.
func (b *builder) boundaries(p Params) {
	byName := map[string]Wall{}
	for _, w := range p.Walls {
		byName[w.Name] = w
	}
	n := 0
	bound := func(space string, elem ifc.Ref, external bool, name string) {
		n++
		side := ifc.Enum("INTERNAL")
		if external {
			side = ifc.Enum("EXTERNAL")
		}
		b.f.Add("IFCRELSPACEBOUNDARY", ifc.GUID(fmt.Sprintf("bound-%d", n)), b.owner,
			name, ifc.Null{}, b.spaces[space], elem, ifc.Null{},
			ifc.Enum("PHYSICAL"), side)
	}

	for _, s := range p.Spaces {
		for _, w := range p.Walls {
			// A room can meet the same wall in more than one stretch: the open
			// living space touches the back wall both at the kitchen end and
			// again at the hallway nook.
			runs := sharedEdges(s.Polygon, w)
			if len(runs) == 0 {
				continue
			}
			bound(s.Name, b.named[w.Name], !w.Interior, s.Name+" / "+w.Name)

			// A window or door in one of those stretches bounds the room too,
			// so a door between two rooms ends up bounding both of them.
			for _, fl := range b.fills {
				if fl.wall != w.Name {
					continue
				}
				for _, run := range runs {
					if fl.at+fl.width/2 > run[0]+tol && fl.at-fl.width/2 < run[1]-tol {
						bound(s.Name, fl.ref, !w.Interior, s.Name+" opening")
						break
					}
				}
			}
		}

		// Below every room is the slab; above it is whichever roof covers it.
		if slab, ok := b.named["Floor slab"]; ok {
			bound(s.Name, slab, true, s.Name+" / floor")
		}
		y1 := p.RiseLength - p.BaseWidth
		lo, hi := yRange(s.Polygon)
		if hi > y1+tol {
			bound(s.Name, b.named["Roof over base"], true, s.Name+" / roof over base")
		}
		if lo < y1-tol {
			bound(s.Name, b.named["Roof over rise"], true, s.Name+" / roof over rise")
		}
	}
}

// sharedEdges returns every stretch, along the wall's own axis, where a face of
// the wall lies on an edge of the polygon.
func sharedEdges(poly [][2]float64, w Wall) [][2]float64 {
	var out [][2]float64
	horizontal := w.Y1 == w.Y2
	for i := range poly {
		a, c := poly[i], poly[(i+1)%len(poly)]
		var lo, hi float64

		switch {
		case horizontal && a[1] == c[1]:
			if math.Abs(w.Y1-w.Thickness/2-a[1]) > tol && math.Abs(w.Y1+w.Thickness/2-a[1]) > tol {
				continue
			}
			lo = math.Max(math.Min(a[0], c[0]), math.Min(w.X1, w.X2))
			hi = math.Min(math.Max(a[0], c[0]), math.Max(w.X1, w.X2))
		case !horizontal && a[0] == c[0]:
			if math.Abs(w.X1-w.Thickness/2-a[0]) > tol && math.Abs(w.X1+w.Thickness/2-a[0]) > tol {
				continue
			}
			lo = math.Max(math.Min(a[1], c[1]), math.Min(w.Y1, w.Y2))
			hi = math.Min(math.Max(a[1], c[1]), math.Max(w.Y1, w.Y2))
		default:
			continue
		}
		if hi-lo > tol {
			out = append(out, [2]float64{lo, hi})
		}
	}
	return out
}

func yRange(poly [][2]float64) (lo, hi float64) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, pt := range poly {
		lo = math.Min(lo, pt[1])
		hi = math.Max(hi, pt[1])
	}
	return lo, hi
}

func inPolygon(poly [][2]float64, x, y float64) bool {
	in := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		xi, yi := poly[i][0], poly[i][1]
		xj, yj := poly[j][0], poly[j][1]
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			in = !in
		}
	}
	return in
}
