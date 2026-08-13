// Package cottage describes the cottage as data and turns it into an IFC4
// model. Everything that could be wrong lives in Default below, so a
// correction is an edit to one number rather than a change to the builder.
package cottage

// Plan coordinates. The model is axis aligned and measured in millimetres from
// the inside face of the west wall (X) and the inside face of the rise's south
// wall (Y). Z is zero at the finished floor. True north is carried in the IFC
// geometric context rather than by rotating the model, which is the usual
// convention.
//
//	y2 = 8600  +----------------------------+   <- back wall, faces NNE
//	           | kitchen | bedroom |h| bath |
//	           |         +---------+-+------+
//	           |                          |
//	y1 = 4190  +---------------+          |
//	                           |  rise    |
//	y0 =    0                  +----------+
//	           x0=0        x1=4340     x2=7940
type Params struct {
	Name    string
	Address string

	// Envelope, inside faces.
	BaseLength float64 // along the back wall, west to east
	BaseWidth  float64 // depth of the base leg, front to back
	RiseLength float64 // total north-south depth, back wall to the rise's south wall
	RiseWidth  float64 // width of the rise leg, west to east

	// Construction.
	ExtWall float64 // exterior wall thickness
	IntWall float64 // interior partition thickness
	Ceiling float64 // finished floor to underside of ceiling
	Slab    float64 // floor slab thickness, sitting below z = 0

	// The photographs show two roofs: a low hipped one over the base leg and a
	// taller, steeper gable over the rise, whose ridge stands well above it.
	BaseEave  float64 // wall top of the base leg
	BasePitch float64 // degrees
	RiseEave  float64 // wall top of the rise leg
	RisePitch float64 // degrees
	Eave      float64 // roof overhang past the outside wall face, all sides

	// Facing is the compass bearing that the back wall looks along, in degrees
	// clockwise from true north. NNE is 22.5.
	Facing float64

	Walls    []Wall
	Openings []Opening
	Spaces   []Space
}

// Wall is a straight wall segment given by the centreline endpoints of its
// finished extent. Segments are written start to end, always running west to
// east or south to north, so that "distance along the wall" is unambiguous.
type Wall struct {
	Name      string
	X1, Y1    float64
	X2, Y2    float64
	Thickness float64
	Height    float64 // wall top; zero falls back to the base leg's eave
	Interior  bool
}

// Opening is a window or a door cut through a wall. At is the world X of the
// opening centre for a wall running west to east, or the world Y for a wall
// running south to north; that reads more directly off a plan than a distance
// from one end.
type Opening struct {
	Wall   string
	Name   string
	Door   bool
	At     float64
	Width  float64
	Height float64
	Sill   float64 // above finished floor; zero for a door
}

// Space is a room, given by its polygon in plan.
type Space struct {
	Name    string
	Long    string
	Polygon [][2]float64
}

// Default is the cottage as described so far. Values marked ASSUMED were not
// given and are placeholders.
func Default() Params {
	const (
		baseLength = 7940.0
		baseWidth  = 4410.0
		riseLength = 8600.0
		riseWidth  = 3600.0

		ext = 170.0 // ASSUMED exterior wall thickness
		in  = 95.0  // ASSUMED partition thickness
		he  = ext / 2
		hi  = in / 2

		baseEave = 2200.0 // ASSUMED, scaled off the photographs
		riseEave = 2500.0 // ASSUMED, the rise clearly stands taller

		// Inside faces of the envelope.
		x0 = 0.0
		x1 = baseLength - riseWidth // 4340, inside face of the rise's west wall
		x2 = baseLength             // 7940
		y0 = 0.0                    // inside face of the rise's south wall
		y1 = riseLength - baseWidth // 4190, inside face of the base's front wall
		y2 = riseLength             // 8600, inside face of the back wall

		// Room divisions along the back wall, east to west. The four clear
		// widths given (1900 + 1000 + 2690 + 2350) sum to exactly 7940, so each
		// partition is centred on the boundary and takes 47.5 mm off each of
		// the two rooms it separates.
		bathW = x2 - 1900 // 6040, bathroom west face
		hallW = bathW - 1000
		bedW  = hallW - 2690 // 2350, bedroom west face = kitchen east face
		bathS = y2 - 1500    // 7100, bathroom south face
		bedS  = y2 - 2130    // 6470, bedroom south face
	)

	return Params{
		Name:    "Cottage",
		Address: "",

		BaseLength: baseLength,
		BaseWidth:  baseWidth,
		RiseLength: riseLength,
		RiseWidth:  riseWidth,

		ExtWall: ext,
		IntWall: in,
		Ceiling: 2200, // ASSUMED
		Slab:    150,  // ASSUMED

		// Eave heights and pitches are scaled off the photographs, not measured.
		BaseEave:  2200,
		BasePitch: 18,
		RiseEave:  2500,
		RisePitch: 30,
		Eave:      400,
		Facing:    22.5, // back wall faces NNE

		// Exterior walls run along the centreline and are extended half a
		// thickness at each end so the corners close. The east wall is written
		// as two segments because the two legs meet it at different heights.
		Walls: []Wall{
			{Name: "Back", X1: -ext, Y1: y2 + he, X2: x2 + ext, Y2: y2 + he, Thickness: ext, Height: baseEave},
			{Name: "Front", X1: -ext, Y1: y1 - he, X2: x1, Y2: y1 - he, Thickness: ext, Height: baseEave},
			{Name: "West", X1: x0 - he, Y1: y1 - ext, X2: x0 - he, Y2: y2 + ext, Thickness: ext, Height: baseEave},
			{Name: "EastBase", X1: x2 + he, Y1: y1 - ext, X2: x2 + he, Y2: y2 + ext, Thickness: ext, Height: baseEave},

			{Name: "RiseSouth", X1: x1 - ext, Y1: y0 - he, X2: x2 + ext, Y2: y0 - he, Thickness: ext, Height: riseEave},
			{Name: "RiseWest", X1: x1 - he, Y1: y0 - ext, X2: x1 - he, Y2: y1, Thickness: ext, Height: riseEave},
			{Name: "EastRise", X1: x2 + he, Y1: y0 - ext, X2: x2 + he, Y2: y1, Thickness: ext, Height: riseEave},

			// Partitions.
			{Name: "BedroomSouth", X1: bedW - hi, Y1: bedS, X2: hallW + hi, Y2: bedS, Thickness: in, Interior: true},
			{Name: "BedroomWest", X1: bedW, Y1: bedS, X2: bedW, Y2: y2, Thickness: in, Interior: true},
			{Name: "BedroomEast", X1: hallW, Y1: bedS, X2: hallW, Y2: y2, Thickness: in, Interior: true},
			{Name: "BathSouth", X1: bathW - hi, Y1: bathS, X2: x2, Y2: bathS, Thickness: in, Interior: true},
			{Name: "BathWest", X1: bathW, Y1: bathS, X2: bathW, Y2: y2, Thickness: in, Interior: true},
		},

		// Sizes are scaled off the photographs against the 900 mm entrance door
		// and the 3940 mm width of the rise's gable, so treat them as within
		// 100 mm or so. Positions on the back wall are ASSUMED: no photograph
		// of that side was supplied.
		Openings: []Opening{
			// Back wall: one window per back room.
			{Wall: "Back", Name: "Kitchen window (back)", At: 1175, Width: 1000, Height: 1000, Sill: 1100},
			{Wall: "Back", Name: "Bedroom window", At: 3695, Width: 1200, Height: 1200, Sill: 900},
			{Wall: "Back", Name: "Bathroom window", At: 6990, Width: 600, Height: 600, Sill: 1500},

			// Front of the base: the wide three-light kitchen window at the west
			// end, then the blue entrance door at the inner corner.
			{Wall: "Front", Name: "Kitchen window (front)", At: 1250, Width: 1800, Height: 1000, Sill: 1000},
			{Wall: "Front", Name: "Entrance door", Door: true, At: 3590, Width: 900, Height: 2040},

			// One wide window centred in the rise's south gable: the dining room.
			{Wall: "RiseSouth", Name: "Dining room window", At: 6140, Width: 1800, Height: 1200, Sill: 900},

			// One in the middle of each side of the rise.
			{Wall: "RiseWest", Name: "Living window (west)", At: 2095, Width: 1200, Height: 1200, Sill: 900},
			{Wall: "EastRise", Name: "Living window (east)", At: 2095, Width: 1200, Height: 1200, Sill: 900},

			// Interior doors, both swinging into the hallway nook.
			{Wall: "BedroomEast", Name: "Bedroom door", Door: true, At: 7900, Width: 800, Height: 2040},
			{Wall: "BathWest", Name: "Bathroom door", Door: true, At: 7850, Width: 700, Height: 2040},
		},

		Spaces: []Space{
			{Name: "Bedroom", Long: "Bedroom", Polygon: [][2]float64{
				{bedW + hi, bedS + hi}, {hallW - hi, bedS + hi}, {hallW - hi, y2}, {bedW + hi, y2},
			}},
			{Name: "Bathroom", Long: "Bathroom", Polygon: [][2]float64{
				{bathW + hi, bathS + hi}, {x2, bathS + hi}, {x2, y2}, {bathW + hi, y2},
			}},
			// Kitchen, dining and living are one open space; only the bedroom
			// and bathroom were described as closed rooms.
			{Name: "Living", Long: "Kitchen / dining / living", Polygon: [][2]float64{
				{x0, y1}, {x1, y1}, {x1, y0}, {x2, y0},
				{x2, bathS - hi}, {bathW - hi, bathS - hi}, {bathW - hi, y2},
				{hallW + hi, y2}, {hallW + hi, bedS - hi}, {bedW - hi, bedS - hi},
				{bedW - hi, y2}, {x0, y2},
			}},
		},
	}
}
