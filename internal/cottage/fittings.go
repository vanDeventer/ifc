package cottage

import (
	"fmt"
	"sort"

	"github.com/vanDeventer/ifc/internal/ifc"
)

// ifcClass maps a fitting kind onto the IFC entity that describes it and the
// value of that entity's PredefinedType. An empty type means the entity has no
// such attribute.
func ifcClass(f Fitting) (entity string, predefined string) {
	switch f.Kind {
	case "sink":
		return "IFCSANITARYTERMINAL", "SINK"
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
	case "counter", "mast":
		return "IFCFURNISHINGELEMENT", ""
	}
	panic(fmt.Sprintf("cottage: unknown fitting kind %q on %q", f.Kind, f.Name))
}

// fittings writes the kitchen units, the heating and the weather station, then
// gathers the devices into one IfcSystem per mbaigo system that owns them.
func (b *builder) fittings(p Params) {
	members := map[string][]ifc.Ref{}

	for _, ft := range p.Fittings {
		entity, predefined := ifcClass(ft)

		cx, cy := (ft.X1+ft.X2)/2, (ft.Y1+ft.Y2)/2
		prof := b.f.RectProfile(ft.Name, 0, 0, ft.X2-ft.X1, ft.Y2-ft.Y1)
		solid := b.f.ExtrudeUp(prof, 0, ft.Top-ft.Base)
		shape := b.f.BodyShape(b.body, solid)
		pl := b.f.PlacedAt(b.floor, cx, cy, ft.Base)

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
	}

	// One IfcSystem per owning mbaigo system, in a stable order.
	names := make([]string, 0, len(members))
	for n := range members {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		sys := b.f.Add("IFCSYSTEM", ifc.GUID("system-"+name), b.owner, name,
			systemDescription(name), "mbaigo")
		b.f.Add("IFCRELASSIGNSTOGROUP", ifc.GUID("assign-"+name), b.owner,
			ifc.Null{}, ifc.Null{}, members[name], ifc.Null{}, sys)
		b.f.Add("IFCRELSERVICESBUILDINGS", ifc.GUID("serves-"+name), b.owner,
			ifc.Null{}, ifc.Null{}, sys, []ifc.Ref{b.building})
	}
}

func systemDescription(name string) any {
	switch name {
	case "BeeKeeper":
		return "mbaigo system switching the heaters through Aqara plugs"
	case "Meteorologue":
		return "mbaigo system reading the NetAtmo weather station"
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
