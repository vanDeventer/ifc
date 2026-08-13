package ifc

// The helpers here cover the small slice of IFC geometry the cottage needs:
// cartesian points, directions, placements and rectangular or arbitrary
// profiles swept along a direction.

// Point3 adds an IfcCartesianPoint in space.
func (f *File) Point3(x, y, z float64) Ref {
	return f.Add("IFCCARTESIANPOINT", []float64{x, y, z})
}

// Point2 adds an IfcCartesianPoint in a profile plane.
func (f *File) Point2(x, y float64) Ref {
	return f.Add("IFCCARTESIANPOINT", []float64{x, y})
}

// Dir3 adds an IfcDirection in space.
func (f *File) Dir3(x, y, z float64) Ref {
	return f.Add("IFCDIRECTION", []float64{x, y, z})
}

// Dir2 adds an IfcDirection in a plane.
func (f *File) Dir2(x, y float64) Ref {
	return f.Add("IFCDIRECTION", []float64{x, y})
}

// Axis3 adds an IfcAxis2Placement3D. A zero axis or refDir is written as unset,
// which IFC reads as the defaults +Z and +X.
func (f *File) Axis3(loc, axis, refDir Ref) Ref {
	return f.Add("IFCAXIS2PLACEMENT3D", loc, axis, refDir)
}

// AxisAt is the common case: an axis placement at a point, unrotated.
func (f *File) AxisAt(x, y, z float64) Ref {
	return f.Axis3(f.Point3(x, y, z), 0, 0)
}

// Axis2 adds an IfcAxis2Placement2D.
func (f *File) Axis2(loc, refDir Ref) Ref {
	return f.Add("IFCAXIS2PLACEMENT2D", loc, refDir)
}

// Placement adds an IfcLocalPlacement. parent may be zero for a placement
// given directly in world coordinates.
func (f *File) Placement(parent, rel Ref) Ref {
	return f.Add("IFCLOCALPLACEMENT", parent, rel)
}

// PlacedAt adds a local placement at a point relative to parent, unrotated.
func (f *File) PlacedAt(parent Ref, x, y, z float64) Ref {
	return f.Placement(parent, f.AxisAt(x, y, z))
}

// PlacedAlong adds a local placement at a point whose local +X axis points
// along (dx, dy) in the horizontal plane. Walls use this so that "along the
// wall" is the local X direction.
func (f *File) PlacedAlong(parent Ref, x, y, z, dx, dy float64) Ref {
	return f.Placement(parent, f.Axis3(f.Point3(x, y, z), f.Dir3(0, 0, 1), f.Dir3(dx, dy, 0)))
}

// RectProfile adds an IfcRectangleProfileDef centred on the origin of the
// profile plane, offset by (cx, cy).
func (f *File) RectProfile(name string, cx, cy, xDim, yDim float64) Ref {
	pos := f.Axis2(f.Point2(cx, cy), 0)
	return f.Add("IFCRECTANGLEPROFILEDEF", Enum("AREA"), name, pos, xDim, yDim)
}

// PolyProfile adds an IfcArbitraryClosedProfileDef from a polygon given as
// x, y pairs. The polygon is closed automatically.
func (f *File) PolyProfile(name string, pts [][2]float64) Ref {
	refs := make([]Ref, 0, len(pts)+1)
	for _, p := range pts {
		refs = append(refs, f.Point2(p[0], p[1]))
	}
	refs = append(refs, refs[0])
	line := f.Add("IFCPOLYLINE", refs)
	return f.Add("IFCARBITRARYCLOSEDPROFILEDEF", Enum("AREA"), name, line)
}

// Extrude sweeps a profile by depth. pos places the profile in the object's
// own coordinate system; axis and refDir orient that system, and may be zero
// for the default of sweeping straight up.
func (f *File) Extrude(profile, pos Ref, dx, dy, dz, depth float64) Ref {
	return f.Add("IFCEXTRUDEDAREASOLID", profile, pos, f.Dir3(dx, dy, dz), depth)
}

// ExtrudeUp is the common case: sweep a profile vertically from height z.
func (f *File) ExtrudeUp(profile Ref, z, depth float64) Ref {
	return f.Extrude(profile, f.AxisAt(0, 0, z), 0, 0, 1, depth)
}

// Brep adds an IfcFacetedBrep from a list of planar faces, each given as a ring
// of points wound counter-clockwise seen from outside the solid. Shapes that
// are not a single sweep, such as a hipped roof, need this rather than an
// extrusion.
func (f *File) Brep(faces [][][3]float64) Ref {
	// One cartesian point per distinct vertex, so the shell shares them.
	seen := map[[3]float64]Ref{}
	point := func(p [3]float64) Ref {
		if r, ok := seen[p]; ok {
			return r
		}
		r := f.Point3(p[0], p[1], p[2])
		seen[p] = r
		return r
	}
	refs := make([]Ref, 0, len(faces))
	for _, face := range faces {
		pts := make([]Ref, len(face))
		for i, p := range face {
			pts[i] = point(p)
		}
		loop := f.Add("IFCPOLYLOOP", pts)
		bound := f.Add("IFCFACEOUTERBOUND", loop, true)
		refs = append(refs, f.Add("IFCFACE", []Ref{bound}))
	}
	return f.Add("IFCFACETEDBREP", f.Add("IFCCLOSEDSHELL", refs))
}

// BodyShape wraps swept solids in the Body representation of a product.
func (f *File) BodyShape(ctx Ref, solids ...Ref) Ref {
	return f.BodyShapeOf(ctx, "SweptSolid", solids...)
}

// BodyShapeOf is BodyShape with an explicit representation type, such as Brep.
func (f *File) BodyShapeOf(ctx Ref, repType string, items ...Ref) Ref {
	rep := f.Add("IFCSHAPEREPRESENTATION", ctx, "Body", repType, items)
	return f.Add("IFCPRODUCTDEFINITIONSHAPE", Null{}, Null{}, []Ref{rep})
}
