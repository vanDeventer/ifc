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

	Walls        []Wall
	Openings     []Opening
	Spaces       []Space
	Fittings     []Fitting
	Penetrations []Penetration
}

// Penetration is a hole cut through a named element by something passing
// through it, such as the flue through the roof overhang. Unlike an Opening it
// is not filled by a door or a window.
type Penetration struct {
	Name           string
	Host           string // the name of the wall or roof it passes through
	X1, Y1, X2, Y2 float64
	Base, Top      float64
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

// Fitting is anything standing in or on the building rather than forming it:
// kitchen units, heaters, the sockets that switch them, and sensors. Each is
// given as its box in plan plus the range of heights it occupies, which is
// enough for both the IFC and the drawing.
type Fitting struct {
	Name           string
	Kind           string // counter, sink, cooker, fridge, heater, plug, sensor, mast, pipe, duct, pit
	X1, Y1, X2, Y2 float64
	Base, Top      float64

	// A pipe or duct is a run rather than a box: a centreline from From to To
	// and a diameter. Set Dia to use these instead of the box above.
	From, To [3]float64
	Dia      float64

	// System names the mbaigo system that owns the device, and is written into
	// the model as an IfcSystem the device is assigned to.
	System string

	// Networks lists the building distribution systems the fitting belongs to,
	// such as the cold water supply. A fixture is usually on more than one.
	Networks []string

	Watts  float64 // heaters
	Sensor string  // IfcSensorTypeEnum member, for sensors
	Note   string
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
			// Not centred in the bathroom: the shower stands in the corner of
			// the two outside walls, so the window sits in what is left of the
			// back wall west of it, with the basin under it.
			{Wall: "Back", Name: "Bathroom window", At: 6564, Width: 600, Height: 600, Sill: 1500},

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

		Fittings: fittings(bedW-hi, bathW+hi, bathS+hi, x1, x2, y1, y2),

		Penetrations: []Penetration{
			{Name: "Flue through the roof overhang", Host: "Roof over base",
				X1: 8175, Y1: 7400, X2: 8325, Y2: 7550, Base: 1900, Top: 2800},
		},
	}
}

// fittings lists the kitchen units, the plumbing and the two mbaigo-controlled
// device families. The arguments are the room faces they are measured against:
// the kitchen's east wall, the bathroom's west and south walls, and the
// envelope's inside faces.
func fittings(kitchenE, bathW, bathS, x1, x2, y1, y2 float64) []Fitting {
	const (
		counter   = 600.0 // depth of the kitchen units
		worktop   = 900.0
		appliance = 600.0 // width of the cooker and of the fridge
		panel     = 110.0 // depth of an electric panel heater
		heaterZ0  = 150.0
		heaterZ1  = 550.0

		// The west-wall run, measured from the back wall southwards: worktop,
		// then the cooker, then the fridge.
		runStart = 1400.0 // "the stove 1400 mm from the base"
		windX    = 3970.0 // roughly the middle of the front elevation
		windDist = 50000.0

		// Pipework. The two water mains run side by side along the inside of
		// the back wall, low down; sizes are nominal.
		main    = 28.0
		branch  = 22.0
		waste   = 75.0
		flue    = 110.0
		eaveTop = 2200.0 // the base leg's wall top, which the flue clears by 1 m
		coldY   = 8555.0 // hard against the back wall
		hotY    = 8480.0 // parallel to it, 75 mm in front
		runZ    = 325.0

		// The greywater pit, 10 m off the north-east corner of the building
		// along the diagonal.
		pitX = 8110 + 7071.0
		pitY = 8770 + 7071.0
	)
	cookerN := y2 - runStart
	cookerS := cookerN - appliance
	fridgeS := cookerS - appliance

	return []Fitting{
		// Kitchen. The sink sits in the run along the back wall, under the
		// window; the cooker and the full-height fridge stand against the west
		// wall, which is the outside wall opposite the bedroom.
		{Name: "Kitchen worktop (back wall)", Kind: "counter",
			X1: 0, Y1: y2 - counter, X2: kitchenE, Y2: y2, Base: 0, Top: worktop},
		{Name: "Kitchen worktop (west wall)", Kind: "counter",
			X1: 0, Y1: cookerN, X2: counter, Y2: y2, Base: 0, Top: worktop},
		{Name: "Kitchen sink", Kind: "sink",
			Networks: []string{"Cold water", "Hot water", "Greywater"},
			X1:       775, Y1: y2 - 540, X2: 1575, Y2: y2 - 60, Base: 820, Top: worktop},
		{Name: "Cooker", Kind: "cooker",
			X1: 0, Y1: cookerS, X2: counter, Y2: cookerN, Base: 0, Top: worktop},
		{Name: "Fridge", Kind: "fridge",
			X1: 0, Y1: fridgeS, X2: counter, Y2: cookerS, Base: 0, Top: 1900,
			Note: "Full height; should finish level with the bedroom wall"},

		// Heating: three 2000 W panels under windows and a 1000 W panel in the
		// bathroom, each switched by an Aqara plug that BeeKeeper drives.
		{Name: "Heater, kitchen", Kind: "heater", System: "BeeKeeper", Watts: 2000,
			X1: 750, Y1: y1, X2: 1750, Y2: y1 + panel, Base: heaterZ0, Top: heaterZ1,
			Note: "Under the kitchen window beside the entrance"},
		{Name: "Plug, kitchen heater", Kind: "plug", System: "BeeKeeper",
			X1: 1950, Y1: y1, X2: 2020, Y2: y1 + 45, Base: 200, Top: 310},

		{Name: "Heater, living west", Kind: "heater", System: "BeeKeeper", Watts: 2000,
			X1: x1, Y1: 1595, X2: x1 + panel, Y2: 2595, Base: heaterZ0, Top: heaterZ1,
			Note: "Under the window on the far side of the entrance"},
		{Name: "Plug, living west heater", Kind: "plug", System: "BeeKeeper",
			X1: x1, Y1: 2795, X2: x1 + 45, Y2: 2865, Base: 200, Top: 310},

		{Name: "Heater, living east", Kind: "heater", System: "BeeKeeper", Watts: 2000,
			X1: x2 - panel, Y1: 1595, X2: x2, Y2: 2595, Base: heaterZ0, Top: heaterZ1,
			Note: "Under the window on the rise's other side"},
		{Name: "Plug, living east heater", Kind: "plug", System: "BeeKeeper",
			X1: x2 - 45, Y1: 2795, X2: x2, Y2: 2865, Base: 200, Top: 310},

		{Name: "Heater, bathroom", Kind: "heater", System: "BeeKeeper", Watts: 1000,
			X1: 6264, Y1: bathS, X2: 6864, Y2: bathS + panel, Base: heaterZ0, Top: heaterZ1,
			Note: "On the wall opposite the bathroom window"},
		{Name: "Plug, bathroom heater", Kind: "plug", System: "BeeKeeper",
			X1: 6964, Y1: bathS, X2: 7034, Y2: bathS + 45, Base: 200, Top: 310},

		// Fixtures. The 30 litre heater is in the hallway between the bathroom
		// and the bedroom, against the outer wall, so all the hot water starts
		// from there.
		{Name: "Water heater, 30 l", Kind: "waterheater",
			Networks: []string{"Cold water", "Hot water"},
			X1:       5340, Y1: 8200, X2: 5740, Y2: y2, Base: 1200, Top: 1800,
			Note: "30 litre electric storage heater; wall-hung, height ASSUMED"},
		{Name: "Bathroom basin", Kind: "basin",
			Networks: []string{"Cold water", "Hot water", "Greywater"},
			X1:       6339, Y1: 8250, X2: 6789, Y2: y2, Base: 780, Top: 900,
			Note: "Under the bathroom window"},
		{Name: "Shower cabin", Kind: "shower",
			Networks: []string{"Cold water", "Hot water", "Greywater"},
			X1:       7140, Y1: 7800, X2: x2, Y2: y2, Base: 0, Top: 2000,
			Note: "In the corner of the two outside walls; 800 x 800 ASSUMED"},

		// The Cinderella burns its waste, so it joins no water network at all,
		// only the flue, which runs up the outside of the wall right behind it.
		{Name: "Toilet, Cinderella Classic", Kind: "toilet", Networks: []string{"Toilet exhaust"},
			X1: 7420, Y1: 7150, X2: x2, Y2: 7800, Base: 0, Top: 600,
			Note: "Electric incinerating toilet: no water supply and no drain. Size ASSUMED"},
		{Name: "Toilet flue, through the east wall", Kind: "duct", Networks: []string{"Toilet exhaust"},
			From: [3]float64{7900, 7475, 520}, To: [3]float64{8250, 7475, 520}, Dia: flue},
		{Name: "Toilet flue, riser outside", Kind: "duct", Networks: []string{"Toilet exhaust"},
			From: [3]float64{8250, 7475, 520}, To: [3]float64{8250, 7475, eaveTop + 1000}, Dia: flue,
			Note: "Terminates 1 m above the eave, passing through the roof overhang"},

		// Cold: in under the kitchen sink, then east along the outer wall and
		// through the bedroom to the heater, the basin and the shower.
		{Name: "Cold water service entry", Kind: "pipe", Networks: []string{"Cold water"},
			From: [3]float64{1175, 8300, -650}, To: [3]float64{1175, 8300, 800}, Dia: main,
			Note: "Enters under the kitchen sink and connects straight to it"},
		{Name: "Cold water main", Kind: "pipe", Networks: []string{"Cold water"},
			From: [3]float64{1175, coldY, runZ}, To: [3]float64{7100, coldY, runZ}, Dia: main,
			Note: "Along the outer wall, through the bedroom"},
		{Name: "Cold branch, water heater", Kind: "pipe", Networks: []string{"Cold water"},
			From: [3]float64{5540, coldY, runZ}, To: [3]float64{5540, coldY, 1300}, Dia: main},
		{Name: "Cold branch, bathroom basin", Kind: "pipe", Networks: []string{"Cold water"},
			From: [3]float64{6564, coldY, runZ}, To: [3]float64{6564, coldY, 760}, Dia: branch},
		{Name: "Cold branch, shower", Kind: "pipe", Networks: []string{"Cold water"},
			From: [3]float64{7100, coldY, runZ}, To: [3]float64{7100, coldY, 1150}, Dia: branch},

		// Hot: out of the heater and back along the same wall, parallel to the
		// cold, reaching both sinks and the shower.
		{Name: "Hot water drop from heater", Kind: "pipe", Networks: []string{"Hot water"},
			From: [3]float64{5540, hotY, 1300}, To: [3]float64{5540, hotY, runZ}, Dia: main},
		{Name: "Hot water main, west to the kitchen", Kind: "pipe", Networks: []string{"Hot water"},
			From: [3]float64{5540, hotY, runZ}, To: [3]float64{1175, hotY, runZ}, Dia: main},
		{Name: "Hot water main, east to the bathroom", Kind: "pipe", Networks: []string{"Hot water"},
			From: [3]float64{5540, hotY, runZ}, To: [3]float64{7100, hotY, runZ}, Dia: main},
		{Name: "Hot branch, kitchen sink", Kind: "pipe", Networks: []string{"Hot water"},
			From: [3]float64{1175, hotY, runZ}, To: [3]float64{1175, hotY, 800}, Dia: branch},
		{Name: "Hot branch, bathroom basin", Kind: "pipe", Networks: []string{"Hot water"},
			From: [3]float64{6564, hotY, runZ}, To: [3]float64{6564, hotY, 760}, Dia: branch},
		{Name: "Hot branch, shower", Kind: "pipe", Networks: []string{"Hot water"},
			From: [3]float64{7100, hotY, runZ}, To: [3]float64{7100, hotY, 1150}, Dia: branch},

		// Waste: one drain under each sink, both buried out to the same pit.
		{Name: "Kitchen sink drain", Kind: "pipe", Networks: []string{"Greywater"},
			From: [3]float64{1350, 8300, 800}, To: [3]float64{1350, 8300, -650}, Dia: waste},
		{Name: "Kitchen drain to the pit", Kind: "pipe", Networks: []string{"Greywater"},
			From: [3]float64{1350, 8300, -650}, To: [3]float64{pitX, pitY, -1100}, Dia: waste},
		{Name: "Bathroom basin drain", Kind: "pipe", Networks: []string{"Greywater"},
			From: [3]float64{6564, 8350, 760}, To: [3]float64{6564, 8350, -650}, Dia: waste},
		{Name: "Shower drain", Kind: "pipe", Networks: []string{"Greywater"},
			From: [3]float64{7540, 8200, 0}, To: [3]float64{7540, 8200, -650}, Dia: waste},
		{Name: "Shower drain, joining the bathroom drain", Kind: "pipe", Networks: []string{"Greywater"},
			From: [3]float64{7540, 8200, -650}, To: [3]float64{6564, 8350, -650}, Dia: waste},
		{Name: "Bathroom drain to the pit", Kind: "pipe", Networks: []string{"Greywater"},
			From: [3]float64{6564, 8350, -650}, To: [3]float64{pitX, pitY, -1100}, Dia: waste},
		{Name: "Greywater pit", Kind: "pit", Networks: []string{"Greywater"},
			X1: pitX - 600, Y1: pitY - 600, X2: pitX + 600, Y2: pitY + 600, Base: -1700, Top: -400,
			Note: "10 m off the north-east corner, taken along the diagonal. Size ASSUMED"},

		// Weather: a NetAtmo station reporting to Meteorologue.
		{Name: "NetAtmo base module", Kind: "sensor", System: "Meteorologue",
			Sensor: "TEMPERATURESENSOR",
			X1:     x1, Y1: 2950, X2: x1 + 60, Y2: 3010, Base: 1500, Top: 1650,
			Note: "Indoor module; position within the living area ASSUMED"},
		{Name: "NetAtmo outdoor module", Kind: "sensor", System: "Meteorologue",
			Sensor: "TEMPERATURESENSOR",
			X1:     3970, Y1: y2 + 170, X2: 4030, Y2: y2 + 215, Base: 1800, Top: 1950,
			Note: "On the north side; height and position along the wall ASSUMED"},
		{Name: "NetAtmo bathroom module", Kind: "sensor", System: "Meteorologue",
			Sensor: "HUMIDITYSENSOR",
			X1:     bathW, Y1: 7900, X2: bathW + 60, Y2: 7960, Base: 1600, Top: 1750,
			Note: "Extra indoor module; position within the bathroom ASSUMED"},
		{Name: "Wind gauge mast", Kind: "mast",
			X1: windX - 40, Y1: -windDist - 40, X2: windX + 40, Y2: -windDist + 40, Base: 0, Top: 3000,
			Note: "Mast height ASSUMED"},
		{Name: "NetAtmo wind gauge", Kind: "sensor", System: "Meteorologue",
			Sensor: "WINDSENSOR",
			X1:     windX - 75, Y1: -windDist - 75, X2: windX + 75, Y2: -windDist + 75, Base: 3000, Top: 3150,
			Note: "50 m in front of the house"},
	}
}
