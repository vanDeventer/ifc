package cottage

import (
	"fmt"
	"math"
	"time"

	"github.com/vanDeventer/ifc/internal/ifc"
)

// builder carries the entities that nearly every product needs.
type builder struct {
	f     *ifc.File
	owner ifc.Ref
	body  ifc.Ref // the Body representation subcontext
	floor ifc.Ref // placement of the storey, and of every element in it

	elements []ifc.Ref // everything contained in the storey
}

// Build turns the parameters into a complete IFC4 model.
func Build(p Params, stamp time.Time) *ifc.File {
	f := ifc.New(p.Name+".ifc", "van Deventer", "Luleå University of Technology", stamp)
	b := &builder{f: f}

	person := f.Add("IFCPERSON", ifc.Null{}, "van Deventer", "Jan", ifc.Null{}, ifc.Null{}, ifc.Null{}, ifc.Null{}, ifc.Null{})
	org := f.Add("IFCORGANIZATION", ifc.Null{}, "Luleå University of Technology", ifc.Null{}, ifc.Null{}, ifc.Null{})
	pao := f.Add("IFCPERSONANDORGANIZATION", person, org, ifc.Null{})
	app := f.Add("IFCAPPLICATION", org, "0.1", "ifcgen", "ifcgen")
	b.owner = f.Add("IFCOWNERHISTORY", pao, app, ifc.Null{}, ifc.Enum("ADDED"),
		ifc.Null{}, ifc.Null{}, ifc.Null{}, int(stamp.Unix()))

	// Millimetres, because every dimension given was in millimetres.
	length := f.Add("IFCSIUNIT", ifc.Star{}, ifc.Enum("LENGTHUNIT"), ifc.Enum("MILLI"), ifc.Enum("METRE"))
	area := f.Add("IFCSIUNIT", ifc.Star{}, ifc.Enum("AREAUNIT"), ifc.Null{}, ifc.Enum("SQUARE_METRE"))
	volume := f.Add("IFCSIUNIT", ifc.Star{}, ifc.Enum("VOLUMEUNIT"), ifc.Null{}, ifc.Enum("CUBIC_METRE"))
	angle := f.Add("IFCSIUNIT", ifc.Star{}, ifc.Enum("PLANEANGLEUNIT"), ifc.Null{}, ifc.Enum("RADIAN"))
	units := f.Add("IFCUNITASSIGNMENT", []ifc.Ref{length, area, volume, angle})

	world := f.AxisAt(0, 0, 0)
	// The model stays axis aligned; the bearing of the back wall is carried as
	// true north, expressed in model coordinates. Model +Y points along that
	// bearing, so north is Facing degrees back the other way.
	rad := p.Facing * math.Pi / 180
	north := f.Dir2(-math.Sin(rad), math.Cos(rad))
	ctx := f.Add("IFCGEOMETRICREPRESENTATIONCONTEXT", ifc.Null{}, "Model", 3, 1e-5, world, north)
	b.body = f.Add("IFCGEOMETRICREPRESENTATIONSUBCONTEXT", "Body", "Model",
		ifc.Star{}, ifc.Star{}, ifc.Star{}, ifc.Star{}, ctx, ifc.Null{}, ifc.Enum("MODEL_VIEW"), ifc.Null{})

	project := f.Add("IFCPROJECT", ifc.GUID("project"), b.owner, p.Name, ifc.Null{},
		ifc.Null{}, ifc.Null{}, ifc.Null{}, []ifc.Ref{ctx}, units)

	sitePl := f.Placement(0, world)
	site := f.Add("IFCSITE", ifc.GUID("site"), b.owner, "Site", ifc.Null{}, ifc.Null{},
		sitePl, ifc.Null{}, ifc.Null{}, ifc.Enum("ELEMENT"), ifc.Null{}, ifc.Null{}, 0.0, ifc.Null{}, ifc.Null{})

	bldgPl := f.PlacedAt(sitePl, 0, 0, 0)
	building := f.Add("IFCBUILDING", ifc.GUID("building"), b.owner, p.Name, ifc.Null{}, ifc.Null{},
		bldgPl, ifc.Null{}, ifc.Null{}, ifc.Enum("ELEMENT"), ifc.Null{}, ifc.Null{}, ifc.Null{})

	b.floor = f.PlacedAt(bldgPl, 0, 0, 0)
	storey := f.Add("IFCBUILDINGSTOREY", ifc.GUID("storey"), b.owner, "Ground floor", ifc.Null{}, ifc.Null{},
		b.floor, ifc.Null{}, ifc.Null{}, ifc.Enum("ELEMENT"), 0.0)

	f.Add("IFCRELAGGREGATES", ifc.GUID("agg-project"), b.owner, ifc.Null{}, ifc.Null{}, project, []ifc.Ref{site})
	f.Add("IFCRELAGGREGATES", ifc.GUID("agg-site"), b.owner, ifc.Null{}, ifc.Null{}, site, []ifc.Ref{building})
	f.Add("IFCRELAGGREGATES", ifc.GUID("agg-building"), b.owner, ifc.Null{}, ifc.Null{}, building, []ifc.Ref{storey})

	b.slab(p)
	b.walls(p)
	b.roof(p)

	f.Add("IFCRELCONTAINEDINSPATIALSTRUCTURE", ifc.GUID("contained"), b.owner,
		ifc.Null{}, ifc.Null{}, b.elements, storey)

	if spaces := b.spaces(p); len(spaces) > 0 {
		f.Add("IFCRELAGGREGATES", ifc.GUID("agg-spaces"), b.owner, ifc.Null{}, ifc.Null{}, storey, spaces)
	}
	return f
}

// outline returns the outside face of the envelope in plan, counter-clockwise.
func outline(p Params) [][2]float64 {
	t := p.ExtWall
	x1 := p.BaseLength - p.RiseWidth
	x2 := p.BaseLength
	y1 := p.RiseLength - p.BaseWidth
	y2 := p.RiseLength
	return [][2]float64{
		{-t, y1 - t}, {x1 - t, y1 - t}, {x1 - t, -t},
		{x2 + t, -t}, {x2 + t, y2 + t}, {-t, y2 + t},
	}
}

func (b *builder) slab(p Params) {
	pl := b.f.PlacedAt(b.floor, 0, 0, -p.Slab)
	prof := b.f.PolyProfile("Floor", outline(p))
	solid := b.f.ExtrudeUp(prof, 0, p.Slab)
	shape := b.f.BodyShape(b.body, solid)
	r := b.f.Add("IFCSLAB", ifc.GUID("slab-floor"), b.owner, "Floor slab", ifc.Null{}, ifc.Null{},
		pl, shape, ifc.Null{}, ifc.Enum("FLOOR"))
	b.elements = append(b.elements, r)
}

func (b *builder) walls(p Params) {
	type placed struct {
		Wall
		ref ifc.Ref
		pl  ifc.Ref
		len float64
	}
	byName := map[string]*placed{}

	for _, w := range p.Walls {
		dx, dy := w.X2-w.X1, w.Y2-w.Y1
		length := math.Hypot(dx, dy)
		ux, uy := dx/length, dy/length

		// The wall's own frame: origin at the start of the centreline, local X
		// along the wall, local Y across it, local Z up. Openings are then
		// placed relative to the wall, which is both correct IFC and the
		// easiest thing for a viewer to interpret.
		height := w.Height
		if height == 0 {
			height = p.BaseEave
		}
		pl := b.f.PlacedAlong(b.floor, w.X1, w.Y1, 0, ux, uy)
		prof := b.f.RectProfile(w.Name, length/2, 0, length, w.Thickness)
		solid := b.f.ExtrudeUp(prof, 0, height)
		shape := b.f.BodyShape(b.body, solid)

		kind := ifc.Enum("SOLIDWALL")
		if w.Interior {
			kind = ifc.Enum("PARTITIONING")
		}
		r := b.f.Add("IFCWALL", ifc.GUID("wall-"+w.Name), b.owner, w.Name, ifc.Null{}, ifc.Null{},
			pl, shape, ifc.Null{}, kind)
		b.elements = append(b.elements, r)
		byName[w.Name] = &placed{Wall: w, ref: r, pl: pl, len: length}
	}

	for i, o := range p.Openings {
		w, ok := byName[o.Wall]
		if !ok {
			panic(fmt.Sprintf("cottage: opening %q names unknown wall %q", o.Name, o.Wall))
		}
		// At is a world X or Y depending on how the wall runs; convert it to a
		// distance along the wall.
		var along float64
		if w.Y1 == w.Y2 {
			along = o.At - w.X1
		} else {
			along = o.At - w.Y1
		}
		if along-o.Width/2 < 0 || along+o.Width/2 > w.len {
			panic(fmt.Sprintf("cottage: opening %q falls outside wall %q", o.Name, o.Wall))
		}

		sill := o.Sill
		if o.Door {
			sill = 0
		}
		seed := fmt.Sprintf("%s-%d", o.Name, i)

		// The void is cut a little proud of both wall faces so that the
		// subtraction is unambiguous.
		const proud = 10.0
		voidPl := b.f.PlacedAt(w.pl, along, 0, sill)
		voidProf := b.f.RectProfile("Void", 0, 0, o.Width, w.Thickness+2*proud)
		voidSolid := b.f.ExtrudeUp(voidProf, 0, o.Height)
		void := b.f.Add("IFCOPENINGELEMENT", ifc.GUID("void-"+seed), b.owner, o.Name+" opening",
			ifc.Null{}, ifc.Null{}, voidPl, b.f.BodyShape(b.body, voidSolid), ifc.Null{}, ifc.Enum("OPENING"))
		b.f.Add("IFCRELVOIDSELEMENT", ifc.GUID("voids-"+seed), b.owner, ifc.Null{}, ifc.Null{}, w.ref, void)

		// The panel that fills it: a thin board on the wall centreline.
		const panel = 60.0
		fillPl := b.f.PlacedAt(w.pl, along, 0, sill)
		fillProf := b.f.RectProfile("Panel", 0, 0, o.Width, panel)
		fillSolid := b.f.ExtrudeUp(fillProf, 0, o.Height)
		fillShape := b.f.BodyShape(b.body, fillSolid)

		var fill ifc.Ref
		if o.Door {
			fill = b.f.Add("IFCDOOR", ifc.GUID("door-"+seed), b.owner, o.Name, ifc.Null{}, ifc.Null{},
				fillPl, fillShape, ifc.Null{}, o.Height, o.Width, ifc.Enum("DOOR"),
				ifc.Enum("SINGLE_SWING_LEFT"), ifc.Null{})
		} else {
			fill = b.f.Add("IFCWINDOW", ifc.GUID("window-"+seed), b.owner, o.Name, ifc.Null{}, ifc.Null{},
				fillPl, fillShape, ifc.Null{}, o.Height, o.Width, ifc.Enum("WINDOW"),
				ifc.Enum("SINGLE_PANEL"), ifc.Null{})
		}
		b.f.Add("IFCRELFILLSELEMENT", ifc.GUID("fills-"+seed), b.owner, ifc.Null{}, ifc.Null{}, void, fill)
		b.elements = append(b.elements, fill)
	}
}

// roof builds the two roofs seen in the photographs: a low hipped one over the
// base leg and a taller gable over the rise, whose ridge stands above it. They
// interpenetrate at the junction, which reads correctly from outside and saves
// modelling the valley; it is massing, not a layered construction.
func (b *builder) roof(p Params) {
	t, e := p.ExtWall, p.Eave
	x1 := p.BaseLength - p.RiseWidth
	x2 := p.BaseLength
	y1 := p.RiseLength - p.BaseWidth
	y2 := p.RiseLength

	// Base leg: hipped at the west end, running east into the rise. The ridge
	// runs along X, halfway across the leg. Modelled as a solid mass, which
	// reads correctly because a hip has no gable to show wall behind it.
	{
		xw := -t             // west wall face
		ys, yn := y1-t, y2+t // front and back wall faces of the base leg
		tan := math.Tan(p.BasePitch * math.Pi / 180)
		h := p.BaseEave
		ym := (ys + yn) / 2
		zr := h + (ym-ys)*tan // ridge
		z0 := h - e*tan       // the overhang carries the slope down past the wall

		xa, xb := xw-e, x2+t+e
		ya, yb := ys-e, yn+e
		xh := xw + (yn-ys)/2 // where the west hip reaches the ridge

		A := [3]float64{xa, ya, z0}
		B := [3]float64{xa, yb, z0}
		C := [3]float64{xb, yb, z0}
		D := [3]float64{xb, ya, z0}
		R1 := [3]float64{xh, ym, zr}
		R2 := [3]float64{xb, ym, zr}

		brep := b.f.Brep([][][3]float64{
			{A, B, C, D},   // eave plane, facing down
			{A, D, R2, R1}, // south slope
			{C, B, R1, R2}, // north slope
			{A, R1, B},     // west hip
			{D, C, R2},     // east end, buried in the rise
		})
		r := b.f.Add("IFCROOF", ifc.GUID("roof-base"), b.owner, "Roof over base", ifc.Null{}, ifc.Null{},
			b.f.PlacedAt(b.floor, 0, 0, 0), b.f.BodyShapeOf(b.body, "Brep", brep), ifc.Null{}, ifc.Enum("HIP_ROOF"))
		b.elements = append(b.elements, r)
	}

	// Rise leg: a gable with its ridge north-south. Here the roof is a covering
	// of finite thickness rather than a solid, so that the gable wall shows
	// through underneath it the way it does in the photographs.
	{
		const thick = 220.0
		wlo, whi := x1-t, x2+t // wall faces
		mid := (wlo + whi) / 2
		tan := math.Tan(p.RisePitch * math.Pi / 180)
		h := p.RiseEave
		zr := h + (mid-wlo)*tan
		lo, hi := wlo-e, whi+e
		z0 := h - e*tan
		ya, yb := -t-e, (y1-t-e+y2+t+e)/2

		// A chevron: along the top of both slopes, then back along the underside.
		prof := b.f.PolyProfile("Roof section", [][2]float64{
			{z0 + thick, lo}, {zr + thick, mid}, {z0 + thick, hi},
			{z0, hi}, {zr, mid}, {z0, lo},
		})
		// Sweep direction +Y: local X is world Z, local Y is world X.
		pos := b.f.Axis3(b.f.Point3(0, ya, 0), b.f.Dir3(0, 1, 0), b.f.Dir3(0, 0, 1))
		solid := b.f.Extrude(prof, pos, 0, 0, 1, yb-ya)
		r := b.f.Add("IFCROOF", ifc.GUID("roof-rise"), b.owner, "Roof over rise", ifc.Null{}, ifc.Null{},
			b.f.PlacedAt(b.floor, 0, 0, 0), b.f.BodyShape(b.body, solid), ifc.Null{}, ifc.Enum("GABLE_ROOF"))
		b.elements = append(b.elements, r)

		// The triangle of boarding between the wall top and the roof.
		gable := b.f.PolyProfile("Gable", [][2]float64{
			{h, wlo}, {zr, mid}, {h, whi},
		})
		gpos := b.f.Axis3(b.f.Point3(0, -t, 0), b.f.Dir3(0, 1, 0), b.f.Dir3(0, 0, 1))
		gsolid := b.f.Extrude(gable, gpos, 0, 0, 1, t)
		g := b.f.Add("IFCWALL", ifc.GUID("gable-rise"), b.owner, "Rise gable", ifc.Null{}, ifc.Null{},
			b.f.PlacedAt(b.floor, 0, 0, 0), b.f.BodyShape(b.body, gsolid), ifc.Null{}, ifc.Enum("SOLIDWALL"))
		b.elements = append(b.elements, g)
	}
}

func (b *builder) spaces(p Params) []ifc.Ref {
	var out []ifc.Ref
	for _, s := range p.Spaces {
		prof := b.f.PolyProfile(s.Name, s.Polygon)
		solid := b.f.ExtrudeUp(prof, 0, p.Ceiling)
		r := b.f.Add("IFCSPACE", ifc.GUID("space-"+s.Name), b.owner, s.Name, ifc.Null{}, ifc.Null{},
			b.f.PlacedAt(b.floor, 0, 0, 0), b.f.BodyShape(b.body, solid), s.Long,
			ifc.Enum("ELEMENT"), ifc.Enum("SPACE"), 0.0)
		out = append(out, r)
	}
	return out
}
