# ifc — the cottage as an IFC4 model

Generates `cottage.ifc` (IFC4, STEP physical file) and `cottage.html`, a
self-contained page that renders it in 3D in a browser. No cgo, no
IfcOpenShell, no JavaScript libraries, nothing fetched from the network.

```
go run ./cmd/ifcgen -out .
open cottage.html
```

Also written: `cottage.fragment.html`, the same viewer as a document body only,
for publishing to a host that supplies its own `<head>`.

## How it works

Go is a **generator**, not a runtime. It runs once, at authoring time, and then
it is out of the picture.

```
  authoring time, in Go                        │  afterwards, in a browser
                                               │
  params.go        build.go        spf.go      │
  ┌─────────┐     ┌─────────┐     ┌─────────┐  │
  │ Params  │────►│ builder │────►│  File   │──┼──► cottage.ifc ──► BIM tools,
  │  data   │     │ entities│     │  STEP   │  │                    mbaigo
  └─────────┘     └─────────┘     └─────────┘  │
                                       │       │
                       viewer.html ◄───┘       │
                            │  the STEP text is│
                            │  pasted into a   │      ┌──────────────────┐
                            ▼  <script> tag    │      │ read the <script>│
                       cottage.html ───────────┼─────►│ parse STEP       │
                                               │      │ placements       │
                                               │      │ profiles → prisms│
                                               │      │ triangles → WebGL│
                                               │      └──────────────────┘
```

Four steps:

1. **`params.go` holds the building as plain data** — a `Params` value listing
   walls, openings, rooms and fittings as numbers.
2. **`build.go` turns that into IFC entities** — walls become `IfcWall` with a
   swept solid, openings become `IfcOpeningElement` plus the relationships that
   void the wall and fill the hole, and so on.
3. **`spf.go` writes those entities out as STEP text.** That is `cottage.ifc`,
   and it is the file any BIM tool or mbaigo system reads.
4. **`viewer.go` pastes that same STEP text into `viewer.html`** at a
   placeholder, producing `cottage.html`.

## Is Go needed afterwards?

**No.** `cottage.html` is a single 77 KB file carrying three things at once: the
model, a reader for it and a renderer. Open it by double-clicking, with no
server, no network and no toolchain, on a machine that has never heard of Go.
Mail it to someone and it works for them too.

That is why the viewer parses the IFC in the page rather than being handed a
pile of triangles. The page is not a picture of the model — it contains the
model, and rebuilds the geometry every time it loads. Deleting every `.go` file
in this repository would not change what `cottage.html` does.

What the browser does on load: split the STEP text into an entity table, walk
each product's `IfcLocalPlacement` chain into a 4×4 matrix, turn each profile
into a polygon, sweep it into a prism, subtract the openings, triangulate, and
hand the result to WebGL. It understands about twenty IFC entity types — the
ones this generator emits — and nothing else.

## Generating another building

Most of the model is already data. These are lists in `Params`, and a different
building is a different list, with no Go to change:

| | |
|---|---|
| `Walls` | any straight segment, with its own thickness and height |
| `Openings` | windows and doors, positioned along a named wall |
| `Spaces` | rooms as polygons |
| `Fittings` | appliances, heaters, sockets, sensors |

Two functions in `build.go` are still written for *this* building and would need
generalising for another:

- `outline` (11 lines) computes the L-shaped footprint from four dimensions.
  A different plan needs a different polygon.
- `roof` (79 lines) builds the specific pair of roofs here — a hip over the base
  leg, a gable over the rise.

Everything else — the whole `internal/ifc` package, the viewer, and the walls,
openings, spaces and fittings code — is indifferent to which building it is
describing. So today: **write a new `Params` value, plus a footprint and a roof.**
Lifting those last two into `Params` as a polygon and a list of roof shapes
would make it data all the way down.

## Files

```
cmd/ifcgen          writes the model and the viewer
internal/ifc        IFC4 writer: entity numbering, STEP values, GUIDs, geometry helpers
internal/cottage    the cottage as data (params.go) and the model builder (build.go)
internal/viewer     the WebGL viewer, embedded with go:embed
pictures/           reference photographs
```

Everything that could be wrong about the building is in `cottage.Default()`.
Correcting a dimension is a one-line edit followed by `go run ./cmd/ifcgen`.

## Coordinates

Millimetres, X east, Y north, Z up, origin at the inside face of the west wall
and the inside face of the rise's south wall. Z zero is the finished floor. The
model stays axis aligned; the NNE bearing of the back wall is carried as
`TrueNorth` on the geometric context, which is the usual convention and keeps
the numbers readable.

```
  y2 = 8600  ╔═══════════╤════════════╤════╤═════════╗   back wall, faces NNE
             ║  kitchen  │  bedroom   │hall│bathroom ║
             ║           │            │    └─────────╢   y = 7100
             ║           │            │              ║
             ║╌╌╌╌╌╌╌╌╌╌╌┴────────────┘              ║   y = 6470
             ║                                       ║
             ║              living / dining          ║
  y1 = 4190  ╚═════════┈┈┈┈════════════╗             ║   entrance in the inner corner
                                       ║             ║
                                       ║    rise     ║
  y0 =    0                            ╚═════════════╝
             x0=0    2350   5040  6040 x1=4340   x2=7940
```

Inside floor area 50.10 m². The four clear widths along the back wall
(1900 + 1000 + 2690 + 2350) sum to exactly 7940, so each partition is centred on
its boundary and takes 47.5 mm off the two rooms it separates.

## Fittings, heating and sensors

Beyond the building itself the model carries the kitchen units, the electric
heating and the weather station, each as its proper IFC entity rather than as
decoration:

| | |
|---|---|
| `IfcSanitaryTerminal` | the sink, in the run under the back window |
| `IfcElectricAppliance` | the cooker and the full-height fridge, against the west wall |
| `IfcSpaceHeater` | three 2000 W panels under windows, one 1000 W in the bathroom |
| `IfcOutlet` | the Aqara plug that switches each heater |
| `IfcSensor` | four NetAtmo modules: indoor, outdoor north, bathroom, and the wind gauge 50 m out |

The two mbaigo systems are in the model as `IfcSystem`, with each device
assigned to its owner and a property set naming it:

```
BeeKeeper     switches the heaters through Aqara plugs
Meteorologue  reads the NetAtmo weather station
```

So a system can find its own devices by walking `IfcRelAssignsToGroup` from the
`IfcSystem` named after it, and read `NominalPower` off each heater, without
knowing anything about this repository's Go types.

One arithmetic conflict is left deliberately visible. The cooker sits 1400 mm
from the back wall as first described, and a 600 cooker followed by a 600 fridge
then ends 420 mm past the bedroom wall rather than level with it. Holding the
fridge to that line instead would put the cooker 980 mm from the back wall.

## What came from where

Given as measurements: the L footprint, the four room widths along the back
wall, the bathroom and bedroom depths, the hallway, the stove position, the
NNE bearing, and which wall each window sits in.

Scaled off the photographs, so within about 100 mm: window and door sizes, eave
heights, roof pitches, the hipped roof over the base leg and the gable over the
rise, and the colours — Falu red boards, white trim, black profiled steel roof,
blue entrance door.

Assumed outright, and worth correcting: wall thicknesses (170 exterior, 95
partitions), the 150 mm slab, the 400 mm eave overhang, and every window
position on the back wall, of which there is no photograph.

The roofs are massing, not layered constructions, and the two interpenetrate at
the junction rather than meeting in a modelled valley.

The file passes `python -m ifcopenshell.validate cottage.ifc --rules` with no
issues, and every product builds geometry under `ifcopenshell.geom`.

## From mbaigo

`internal/cottage` holds no CLI state, so an mbaigo system can import it,
call `cottage.Build`, and serve the model or its geometry as a service.
`internal/ifc` is independent of the cottage and can carry any IFC4 model.
