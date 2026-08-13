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

## Layout

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

## From mbaigo

`internal/cottage` holds no CLI state, so an mbaigo system can import it,
call `cottage.Build`, and serve the model or its geometry as a service.
`internal/ifc` is independent of the cottage and can carry any IFC4 model.
