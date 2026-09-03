# ifc — the cottage as an IFC4 model

Generates three views of one cottage: `cottage.ifc` (IFC4, STEP physical file),
`cottage.html`, a self-contained page that renders it in 3D in a browser, and
`cottage.ttl`, the same model as RDF using the Linked Building Data ontologies.
No cgo, no IfcOpenShell, no JavaScript libraries, nothing fetched from the
network.

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

**No.** `cottage.html` is a single 120 KB file carrying three things at once: the
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
cmd/ifcgen          writes the model, the viewer and the RDF
cmd/ifc2ttl         converts any IFC4 file to Turtle on its own
internal/step       a STEP (ISO 10303-21) reader
internal/lbd        the IFC to Linked Building Data mapping
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

## Fittings, plumbing, heating and sensors

Beyond the building itself the model carries the kitchen units, the plumbing,
the electric heating and the weather station, each as its proper IFC entity
rather than as decoration:

| | |
|---|---|
| `IfcSanitaryTerminal` | kitchen sink, bathroom basin, shower cabin, toilet |
| `IfcElectricAppliance` | cooker, full-height fridge, 30 litre water heater |
| `IfcPipeSegment` | the water mains, their branches and the two buried drains |
| `IfcDistributionChamberElement` | the greywater pit, 10 m off the north-east corner |
| `IfcDuctSegment` | the toilet flue, out through the east wall and up outside |
| `IfcSpaceHeater` | three 2000 W panels under windows, one 1000 W in the bathroom |
| `IfcOutlet` | the Aqara plug that switches each heater |
| `IfcSensor` | four NetAtmo modules: indoor, outdoor north, bathroom, and the wind gauge 50 m out |

Two kinds of system group them. The mbaigo systems are `IfcSystem` — software
that drives devices — and the building's networks are `IfcDistributionSystem`,
which say what is connected to what:

```
BeeKeeper       IfcSystem              switches the heaters through Aqara plugs
Meteorologue    IfcSystem              reads the NetAtmo weather station
Cold water      DOMESTICCOLDWATER      entry, main, branches, heater, fixtures
Hot water       DOMESTICHOTWATER       heater, mains both ways, branches, fixtures
Greywater       WASTEWATER             two sink drains and the pit
Toilet exhaust  EXHAUST                toilet and the two flue segments
```

So a system can find its own devices by walking `IfcRelAssignsToGroup` from the
group named after it, and read `NominalPower` off each heater, without knowing
anything about this repository's Go types.

The Cinderella incinerates rather than flushes, so it joins no water network at
all — only the flue. It is still classified `TOILETPAN`, because that is the
fixture a room schedule looks for; what makes it unusual is in its properties
(`WaterSupply: None`, `WasteDisposal: Incineration, electric`).

The runs are modelled, not just the fixtures. Cold comes in under the kitchen
sink, straight into it, then east along the inside of the back wall and through
the bedroom to the heater, the basin and the shower; hot leaves the heater and
comes back along the same wall, parallel to the cold, 75 mm in front of it. A
drain under each sink drops below the slab and runs out to the pit.

A pipe is given as a centreline and a diameter rather than a box, which is what
lets the drains run diagonally out to the pit. `IfcPipeSegment` and
`IfcDuctSegment` are the only fittings that work that way; everything else is
still an upright box.

In the viewer the runs take their colour from the network that owns them, read
out of the model's own `IfcDistributionSystem` groups rather than from their
names. The **Site** view frames the whole plot and hides the ground, which is
the only way to see the buried drains and the pit.

The shower's waste is not modelled: two drains were described, one under each
sink, and where the shower joins them was not said.

One arithmetic conflict is left deliberately visible. The cooker sits 1400 mm
from the back wall as first described, and a 600 cooker followed by a 600 fridge
then ends 420 mm past the bedroom wall rather than level with it. Holding the
fridge to that line instead would put the cooker 980 mm from the back wall.

## Relationships in the IFC

Geometry alone cannot be asked questions. These four passes add the edges that
can, and none of them touch the geometry code:

**Containment.** A fixture or device is contained in the room it stands in, not
in the storey: the shower is in the Bathroom, the cooker in the Living space.
IFC allows one container per element, so this is a partition — walls, slabs,
roofs, windows, doors and the distribution runs stay with the storey, because a
pipe passes through rooms rather than standing in one.

**Space boundaries.** `IfcRelSpaceBoundary` records which element bounds which
room, and whether that boundary faces outside. Derived from the geometry: a wall
bounds a room when one of its faces lies along an edge of the room's polygon,
and a window or door bounds it when it falls in that stretch of wall. A room can
meet the same wall in two separate stretches — the open living space touches the
back wall at the kitchen end and again at the hallway nook — and both count.

This is what makes a door an edge between two rooms:

```
Bedroom door    Bedroom  <-> Living
Bathroom door   Bathroom <-> Living
Entrance door   Living                 (the other side is outside)
```

**Types.** 68 occurrences over 33 type objects. "The window in the bathroom" is
an occurrence of "Window 600 x 600", of which there is one definition however
many are installed — the individual-and-definition split, via
`IfcRelDefinesByType`.

**Classification.** Attached to the type objects, so occurrences inherit it
through their type. **The identifications are placeholders.** The IFC wiring is
real — `IfcClassification`, `IfcClassificationReference`,
`IfcRelAssociatesClassification` — but the codes are invented, and the scheme is
named so that nobody downstream mistakes them for a published table. Swapping in
CoClass means replacing the source and the ID column in `params.go`; nothing
else changes. A test checks that every code names a product the model actually
builds, since one that does not is silently dropped.

## RDF, with the LBD ontologies

`cottage.ttl` is the same model as Linked Building Data, written by
`cmd/ifc2ttl`, which reads any IFC4 file rather than only this one:

```
go run ./cmd/ifc2ttl -o cottage.ttl cottage.ifc
```

The IFC file stays authoritative for geometry, and says so: the building carries
`omg:hasGeometry` pointing at a node that names the IFC file through
`fog:asIfc_v2x4`. What crosses over is the part a graph can be asked questions
of. 1248 triples over these ontologies:

| | |
|---|---|
| **BOT** | spatial hierarchy, containment, adjacency, hosting |
| **BEO** | `beo:Wall`, `beo:Wall-PARTITIONING`, `beo:Window`, `beo:Slab-FLOOR`, `beo:Roof-HIP_ROOF` |
| **MEP** | `mep:SpaceHeater-CONVECTOR`, `mep:Sensor-WINDSENSOR`, `mep:SanitaryTerminal-TOILETPAN` |
| **FSO** | the four flow systems and the role of each component |
| **BPO** | `bpo:realisesObject`, the installed element to its product |
| **SKOS** | the classification scheme |
| **OMG / FOG** | where the geometry stayed behind |

Every class the converter can emit was checked against the published ontology
rather than recalled, and a test keeps it that way. An IFC entity with no class
in BEO or MEP stays a plain `bot:Element` and the file says so in a comment, in
preference to inventing a class that may not exist. Two terms have no LBD
equivalent and are declared in the file itself: `cot:SoftwareSystem` for the
mbaigo systems, which drive devices rather than carry fluid, and
`cot:boundaryFacing` for the inside/outside flag on a space boundary.

Space boundaries come across twice on purpose: as `bot:adjacentElement` for
querying, and as a reified `bot:Interface` that can carry the flag.

Asking it something, in SPARQL:

```sparql
# Which rooms does each door join?
SELECT ?door (GROUP_CONCAT(?room; separator=" <-> ") AS ?joins) WHERE {
  ?s a bot:Space ; rdfs:label ?room ; bot:adjacentElement ?d .
  ?d a beo:Door ; rdfs:label ?door .
} GROUP BY ?door

#   Bathroom door   Bathroom <-> Living
#   Bedroom door    Bedroom <-> Living
#   Entrance door   Living
```

```sparql
# What does the cold water system feed?
SELECT ?component WHERE {
  ?sys a fso:DistributionSystem ; rdfs:label "Cold water" ; fso:hasComponent ?c .
  ?c rdfs:label ?component .
}
```

## Joining the runtime graph

`cottage.align.ttl` is the third graph, joining mbaigo's runtime graph to this
building model. mbaigo names a device's place with a human label — `alc:Kitchen`
— and the IFC names the same room by its `IfcGloballyUniqueId`. Nothing relates
them, so the join has to be asserted.

**The generator can assert it without anyone maintaining a list**, because the
room identifiers are `SpaceGUID(name)`, a pure function of the room's name in
`params.go`. The generator knows both ends, so it writes the mapping itself. A
test checks that every identifier the alignment names is really in the IFC.

**It is not `owl:sameAs`.** Two of the cloud's functional locations, Kitchen and
Entrance, are in the same open room here. Identity is symmetric and transitive,
so asserting it twice entails `alc:Kitchen owl:sameAs alc:Entrance`, and a
reasoner merges everything said about either. The relation is many functional
locations to one room, which identity cannot express. `cot:locatedIn` says only
what is true.

**It is a separate file** because it speaks about a namespace neither side owns
and is true of one local cloud only. `cottage.ttl` stays good on its own.

Loading all three and asking which room a running device is in:

```sparql
SELECT ?device ?room WHERE {
  ?d afo:hasName ?device ; afo:hasFunctionalLocation ?fl .
  ?fl cot:locatedIn ?space .
  ?space a bot:Space ; rdfs:label ?room .
}
#   thermostat | Living
```

Which then reaches everything the building model knows: what else is in that
room, what bounds it, which flow systems serve it.

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

One of those guesses has since been corrected by the plumbing rather than by a
measurement. The bathroom window was centred in the room until the shower cabin
arrived in the corner of the two outside walls, which is the same corner; the
window has moved west to sit in what is left of that wall, with the basin under
it. Fixtures are a constraint on the openings, not just contents of the room.

The roofs are massing, not layered constructions, and the two interpenetrate at
the junction rather than meeting in a modelled valley.

The file passes `python -m ifcopenshell.validate cottage.ifc --rules` with no
issues, and every product builds geometry under `ifcopenshell.geom`.

## From mbaigo

`internal/cottage` holds no CLI state, so an mbaigo system can import it,
call `cottage.Build`, and serve the model or its geometry as a service.
`internal/ifc` is independent of the cottage and can carry any IFC4 model.
