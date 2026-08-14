package cottage

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/vanDeventer/ifc/internal/ifc"
)

// ifcClass maps a fitting kind onto the IFC entity that describes it and the
// value of that entity's PredefinedType. An empty type means the entity has no
// such attribute.
func ifcClass(f Fitting) (entity string, predefined string) {
	switch f.Kind {
	case "sink":
		return "IFCSANITARYTERMINAL", "SINK"
	case "basin":
		return "IFCSANITARYTERMINAL", "WASHHANDBASIN"
	case "shower":
		return "IFCSANITARYTERMINAL", "SHOWER"
	case "toilet":
		// Classified as a WC even though it burns rather than flushes: it is
		// the fixture a plumber or a room schedule is looking for. What makes
		// it unusual is in its property set instead.
		return "IFCSANITARYTERMINAL", "TOILETPAN"
	case "waterheater":
		return "IFCELECTRICAPPLIANCE", "FREESTANDINGWATERHEATER"
	case "pipe":
		return "IFCPIPESEGMENT", "RIGIDSEGMENT"
	case "duct":
		return "IFCDUCTSEGMENT", "RIGIDSEGMENT"
	case "cooker":
		return "IFCELECTRICAPPLIANCE", "ELECTRICCOOKER"
	case "fridge":
		return "IFCELECTRICAPPLIANCE", "REFRIGERATOR"
	case "heater":
		return "IFCSPACEHEATER", "CONVECTOR"
	case "plug":
		return "IFCOUTLET", "POWEROUTLET"
	case "sensor":
		s := f.Sensor
		if s == "" {
			s = "NOTDEFINED"
		}
		return "IFCSENSOR", s
	case "pit":
		return "IFCDISTRIBUTIONCHAMBERELEMENT", "SUMP"
	case "counter", "mast":
		return "IFCFURNISHINGELEMENT", ""
	}
	panic(fmt.Sprintf("cottage: unknown fitting kind %q on %q", f.Kind, f.Name))
}

// fittings writes the kitchen units, the heating and the weather station, then
// gathers the devices into one IfcSystem per mbaigo system that owns them.
func (b *builder) fittings(p Params) {
	members := map[string][]ifc.Ref{}
	networks := map[string][]ifc.Ref{}

	for _, ft := range p.Fittings {
		entity, predefined := ifcClass(ft)

		var pl, shape ifc.Ref
		if ft.Dia > 0 {
			// A run: sweep the section along the centreline.
			dx, dy, dz := ft.To[0]-ft.From[0], ft.To[1]-ft.From[1], ft.To[2]-ft.From[2]
			length := math.Sqrt(dx*dx + dy*dy + dz*dz)
			if length == 0 {
				panic(fmt.Sprintf("cottage: run %q has zero length", ft.Name))
			}
			pl = b.f.PlacedAlongRun(b.floor, ft.From[0], ft.From[1], ft.From[2],
				dx/length, dy/length, dz/length)
			prof := b.f.RectProfile(ft.Name, 0, 0, ft.Dia, ft.Dia)
			shape = b.f.BodyShape(b.body, b.f.ExtrudeUp(prof, 0, length))
		} else {
			prof := b.f.RectProfile(ft.Name, 0, 0, ft.X2-ft.X1, ft.Y2-ft.Y1)
			shape = b.f.BodyShape(b.body, b.f.ExtrudeUp(prof, 0, ft.Top-ft.Base))
			pl = b.f.PlacedAt(b.floor, (ft.X1+ft.X2)/2, (ft.Y1+ft.Y2)/2, ft.Base)
		}

		args := []any{
			ifc.GUID("fitting-" + ft.Name), b.owner, ft.Name, ifc.Null{},
			ifc.Null{}, pl, shape, ifc.Null{},
		}
		if predefined != "" {
			args = append(args, ifc.Enum(predefined))
		}
		r := b.f.Add(entity, args...)
		b.elements = append(b.elements, r)

		if props := fittingProps(ft); len(props) > 0 {
			b.pset(r, "Pset_Mbaigo_"+psetSuffix(ft), props)
		}
		if ft.System != "" {
			members[ft.System] = append(members[ft.System], r)
		}
		for _, n := range ft.Networks {
			networks[n] = append(networks[n], r)
		}
	}

	// One IfcSystem per owning mbaigo system: software that drives devices.
	for _, name := range sortedKeys(members) {
		sys := b.f.Add("IFCSYSTEM", ifc.GUID("system-"+name), b.owner, name,
			systemDescription(name), "mbaigo")
		b.serves(sys, name, members[name])
	}

	// One IfcDistributionSystem per network: what is physically connected to
	// what, even though the runs between them are not modelled.
	for _, name := range sortedKeys(networks) {
		sys := b.f.Add("IFCDISTRIBUTIONSYSTEM", ifc.GUID("network-"+name), b.owner, name,
			systemDescription(name), ifc.Null{}, ifc.Null{}, ifc.Enum(networkType(name)))
		b.serves(sys, name, networks[name])
	}
}

// serves assigns members to a system and records that it serves the building.
func (b *builder) serves(sys ifc.Ref, name string, members []ifc.Ref) {
	b.f.Add("IFCRELASSIGNSTOGROUP", ifc.GUID("assign-"+name), b.owner,
		ifc.Null{}, ifc.Null{}, members, ifc.Null{}, sys)
	b.f.Add("IFCRELSERVICESBUILDINGS", ifc.GUID("serves-"+name), b.owner,
		ifc.Null{}, ifc.Null{}, sys, []ifc.Ref{b.building})
}

func sortedKeys(m map[string][]ifc.Ref) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// networkType is the IfcDistributionSystemEnum member for a named network.
func networkType(name string) string {
	switch name {
	case "Cold water":
		return "DOMESTICCOLDWATER"
	case "Hot water":
		return "DOMESTICHOTWATER"
	case "Toilet exhaust":
		return "EXHAUST"
	case "Greywater":
		return "WASTEWATER"
	}
	return "NOTDEFINED"
}

func systemDescription(name string) any {
	switch name {
	case "BeeKeeper":
		return "mbaigo system switching the heaters through Aqara plugs"
	case "Meteorologue":
		return "mbaigo system reading the NetAtmo weather station"
	case "Cold water":
		return "Mains cold water, entering under the kitchen sink"
	case "Hot water":
		return "Fed from the 30 litre electric heater in the hallway"
	case "Toilet exhaust":
		return "Flue carrying the incinerating toilet's exhaust outside"
	case "Greywater":
		return "Waste from the two sinks, to a pit 10 m off the north-east corner"
	}
	return ifc.Null{}
}

func psetSuffix(f Fitting) string {
	if f.System != "" {
		return f.System
	}
	return "Fitting"
}

type prop struct {
	name  string
	value ifc.Typed
}

func fittingProps(f Fitting) []prop {
	var out []prop
	if f.System != "" {
		out = append(out, prop{"MbaigoSystem", ifc.Label(f.System)})
	}
	switch f.Kind {
	case "heater":
		out = append(out,
			prop{"NominalPower", ifc.Power(f.Watts)},
			prop{"SwitchedBy", ifc.Label("Aqara smart plug")})
	case "plug":
		out = append(out, prop{"DeviceType", ifc.Label("Aqara smart plug")})
	case "sensor":
		out = append(out, prop{"DeviceType", ifc.Label("NetAtmo module")})
	case "toilet":
		out = append(out,
			prop{"Model", ifc.Label("Cinderella Classic")},
			prop{"WaterSupply", ifc.Label("None")},
			prop{"WasteDisposal", ifc.Label("Incineration, electric")})
	case "waterheater":
		out = append(out, prop{"StorageCapacity", ifc.Volume(0.030)})
	}
	if len(f.Networks) > 0 {
		out = append(out, prop{"Networks", ifc.Label(strings.Join(f.Networks, ", "))})
	}
	if f.Note != "" {
		out = append(out, prop{"Note", ifc.Text(f.Note)})
	}
	return out
}

// pset attaches a property set to one element.
func (b *builder) pset(on ifc.Ref, name string, props []prop) {
	refs := make([]ifc.Ref, len(props))
	for i, p := range props {
		refs[i] = b.f.Add("IFCPROPERTYSINGLEVALUE", p.name, ifc.Null{}, p.value, ifc.Null{})
	}
	set := b.f.Add("IFCPROPERTYSET", ifc.GUID(fmt.Sprintf("pset-%d-%s", on, name)),
		b.owner, name, ifc.Null{}, refs)
	b.f.Add("IFCRELDEFINESBYPROPERTIES", ifc.GUID(fmt.Sprintf("defines-%d-%s", on, name)),
		b.owner, ifc.Null{}, ifc.Null{}, []ifc.Ref{on}, set)
}
