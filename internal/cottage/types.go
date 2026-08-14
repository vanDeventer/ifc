package cottage

import (
	"sort"

	"github.com/vanDeventer/ifc/internal/ifc"
)

// typeEntity is the IfcTypeProduct that goes with an occurrence entity.
func typeEntity(entity string) string {
	switch entity {
	case "IFCWALL":
		return "IFCWALLTYPE"
	case "IFCWINDOW":
		return "IFCWINDOWTYPE"
	case "IFCDOOR":
		return "IFCDOORTYPE"
	case "IFCSLAB":
		return "IFCSLABTYPE"
	case "IFCROOF":
		return "IFCROOFTYPE"
	case "IFCSPACEHEATER":
		return "IFCSPACEHEATERTYPE"
	case "IFCOUTLET":
		return "IFCOUTLETTYPE"
	case "IFCSENSOR":
		return "IFCSENSORTYPE"
	case "IFCSANITARYTERMINAL":
		return "IFCSANITARYTERMINALTYPE"
	case "IFCELECTRICAPPLIANCE":
		return "IFCELECTRICAPPLIANCETYPE"
	case "IFCPIPESEGMENT":
		return "IFCPIPESEGMENTTYPE"
	case "IFCDUCTSEGMENT":
		return "IFCDUCTSEGMENTTYPE"
	case "IFCDISTRIBUTIONCHAMBERELEMENT":
		return "IFCDISTRIBUTIONCHAMBERELEMENTTYPE"
	case "IFCFURNISHINGELEMENT":
		return "IFCFURNISHINGELEMENTTYPE"
	}
	return ""
}

// types writes one type object per distinct product and points every
// occurrence at it. This is the individual-and-definition split: "the window in
// the bathroom" is an occurrence of "Window 600 x 600", of which there is one
// definition however many are installed.
func (b *builder) types() map[string]ifc.Ref {
	group := map[string][]ifc.Ref{}
	entityOf := map[string]string{}
	predefOf := map[string]string{}

	for _, r := range b.records {
		if r.typeName == "" || typeEntity(r.entity) == "" {
			continue
		}
		group[r.typeName] = append(group[r.typeName], r.ref)
		entityOf[r.typeName] = r.entity
		predefOf[r.typeName] = r.predefined
	}

	names := make([]string, 0, len(group))
	for n := range group {
		names = append(names, n)
	}
	sort.Strings(names)

	out := make(map[string]ifc.Ref, len(names))
	for _, name := range names {
		t := b.typeObject(entityOf[name], name, predefOf[name])
		b.f.Add("IFCRELDEFINESBYTYPE", ifc.GUID("definedby-"+name), b.owner,
			ifc.Null{}, ifc.Null{}, group[name], t)
		out[name] = t
	}
	return out
}

func (b *builder) typeObject(entity, name, predefined string) ifc.Ref {
	te := typeEntity(entity)
	// IfcTypeObject, IfcTypeProduct and IfcElementType between them: GlobalId,
	// OwnerHistory, Name, Description, ApplicableOccurrence, HasPropertySets,
	// RepresentationMaps, Tag, ElementType.
	base := []any{ifc.GUID("type-" + name), b.owner, name, ifc.Null{},
		ifc.Null{}, ifc.Null{}, ifc.Null{}, ifc.Null{}, ifc.Null{}}

	switch te {
	case "IFCFURNISHINGELEMENTTYPE":
		return b.f.Add(te, base...) // no PredefinedType on this one
	case "IFCWINDOWTYPE":
		return b.f.Add(te, append(base, ifc.Enum(predefined),
			ifc.Enum("SINGLE_PANEL"), false, ifc.Null{})...)
	case "IFCDOORTYPE":
		return b.f.Add(te, append(base, ifc.Enum(predefined),
			ifc.Enum("SINGLE_SWING_LEFT"), false, ifc.Null{})...)
	default:
		return b.f.Add(te, append(base, ifc.Enum(predefined))...)
	}
}

// classify associates type objects with classification references, so an
// occurrence inherits its classification through its type.
func (b *builder) classify(p Params, types map[string]ifc.Ref) {
	c := p.Classification
	if c.Source == "" || len(c.Codes) == 0 {
		return
	}
	scheme := b.f.Add("IFCCLASSIFICATION", c.Source, c.Edition, ifc.Null{},
		c.Name, c.Description, ifc.Null{}, ifc.Null{})

	for _, code := range c.Codes {
		t, ok := types[code.TypeName]
		if !ok {
			continue // the model no longer has that product
		}
		refr := b.f.Add("IFCCLASSIFICATIONREFERENCE", ifc.Null{}, code.ID, code.Title,
			scheme, ifc.Null{}, ifc.Null{})
		b.f.Add("IFCRELASSOCIATESCLASSIFICATION", ifc.GUID("class-"+code.TypeName),
			b.owner, ifc.Null{}, ifc.Null{}, []ifc.Ref{t}, refr)
	}
}
