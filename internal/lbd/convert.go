// Package lbd converts an IFC model into RDF using the Linked Building Data
// ontologies, and writes it as Turtle.
//
// The IFC file stays the source of truth for geometry. What comes across is the
// part a graph can be asked questions of: the spatial hierarchy, what contains
// and bounds what, which product each installed element realises, the flow
// systems, the classification and the property sets.
package lbd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vanDeventer/ifc/internal/step"
)

// Options configures a conversion.
type Options struct {
	Base   string // instance namespace; must end in / or #
	Source string // the IFC file this came from, recorded as provenance
	Title  string
}

type converter struct {
	f    *step.File
	opt  Options
	inst string
	voc  string
	b    strings.Builder

	elements map[int]*step.Entity // everything contained in a spatial structure
}

// Convert reads a parsed IFC file and returns a Turtle document.
func Convert(f *step.File, opt Options) (string, error) {
	if opt.Base == "" {
		opt.Base = "https://example.org/building/"
	}
	if !strings.HasSuffix(opt.Base, "/") && !strings.HasSuffix(opt.Base, "#") {
		opt.Base += "/"
	}
	c := &converter{f: f, opt: opt, inst: opt.Base, elements: map[int]*step.Entity{}}
	c.voc = opt.Base + "vocab#"

	c.prefixes()
	c.ontology()
	c.localVocabulary()
	c.spatial()
	c.collectElements()
	c.elementsSection()
	c.hosting()
	c.boundaries()
	c.types()
	c.systems()
	c.classification()
	c.properties()
	return c.b.String(), nil
}

func (c *converter) w(format string, a ...any) {
	fmt.Fprintf(&c.b, format+"\n", a...)
}

func (c *converter) head(title string) {
	c.w("")
	c.w("# ---------------------------------------------------------------- %s", title)
}

func (c *converter) prefixes() {
	for _, p := range [][2]string{
		{"inst", c.inst}, {"cot", c.voc},
		{"bot", nsBOT}, {"beo", nsBEO}, {"mep", nsMEP}, {"fso", nsFSO},
		{"bpo", nsBPO}, {"omg", nsOMG}, {"fog", nsFOG},
		{"skos", nsSKOS}, {"dcterms", nsDCTERMS},
		{"rdfs", nsRDFS}, {"owl", nsOWL}, {"xsd", nsXSD},
	} {
		c.w("@prefix %-8s <%s> .", p[0]+":", p[1])
	}
}

func (c *converter) ontology() {
	c.head("this dataset")
	title := c.opt.Title
	if title == "" {
		if p := c.f.Of("IFCPROJECT"); len(p) > 0 {
			title = step.Str(p[0].Arg(2))
		}
	}
	c.w("<%s> a owl:Ontology ;", strings.TrimSuffix(strings.TrimSuffix(c.inst, "/"), "#"))
	if title != "" {
		c.w("    dcterms:title %s ;", lit(title))
	}
	if c.opt.Source != "" {
		c.w("    dcterms:source <%s> ;", c.opt.Source)
	}
	c.w("    rdfs:comment %s .", lit("Converted from IFC4. Geometry is not carried over; "+
		"the IFC file named as dcterms:source remains authoritative for it."))

	// Say where the geometry lives, rather than pretending it is absent.
	if c.opt.Source != "" {
		if bs := c.f.Of("IFCBUILDING"); len(bs) > 0 {
			c.w("")
			c.w("%s omg:hasGeometry inst:geometry .", c.iri(bs[0]))
			c.w("inst:geometry a omg:Geometry ; fog:asIfc_v2x4 <%s> .", c.opt.Source)
		}
	}
}

// localVocabulary declares the few terms LBD has no equivalent for, so the file
// explains itself rather than leaving a reader to guess.
func (c *converter) localVocabulary() {
	c.head("terms with no LBD equivalent")
	c.w("cot:SoftwareSystem a owl:Class ;")
	c.w("    rdfs:label %s ;", lit("Software system"))
	c.w("    rdfs:comment %s .", lit("A running system that drives or reads devices in the "+
		"building, as distinct from a flow system. Here: the mbaigo systems."))
	c.w("cot:manages a owl:ObjectProperty ;")
	c.w("    rdfs:label %s ;", lit("manages"))
	c.w("    rdfs:comment %s .", lit("Relates a software system to a device it drives or reads."))
	c.w("cot:boundaryFacing a owl:DatatypeProperty ;")
	c.w("    rdfs:label %s ;", lit("boundary facing"))
	c.w("    rdfs:comment %s .", lit("INTERNAL or EXTERNAL, from the IFC space boundary."))
}

func (c *converter) spatial() {
	c.head("spatial structure")
	for _, pair := range [][2]string{
		{"IFCSITE", "bot:Site"}, {"IFCBUILDING", "bot:Building"},
		{"IFCBUILDINGSTOREY", "bot:Storey"}, {"IFCSPACE", "bot:Space"},
	} {
		for _, e := range c.f.Of(pair[0]) {
			c.w("%s a %s ;", c.iri(e), pair[1])
			if long := step.Str(e.Arg(7)); long != "" && pair[0] == "IFCSPACE" {
				c.w("    rdfs:comment %s ;", lit(long))
			}
			c.label(e, "    ")
		}
	}

	// IfcRelAggregates carries the hierarchy. BOT has no project, so the site
	// is the root and the project survives only as this dataset's title.
	c.w("")
	for _, r := range c.f.Of("IFCRELAGGREGATES") {
		from := c.f.Get(r.Arg(4))
		if from == nil {
			continue
		}
		for _, m := range step.List(r.Arg(5)) {
			to := c.f.Get(m)
			if to == nil {
				continue
			}
			var pred string
			switch to.Type {
			case "IFCBUILDING":
				pred = "bot:hasBuilding"
			case "IFCBUILDINGSTOREY":
				pred = "bot:hasStorey"
			case "IFCSPACE":
				pred = "bot:hasSpace"
			default:
				continue
			}
			c.w("%s %s %s .", c.iri(from), pred, c.iri(to))
		}
	}
}

// collectElements takes the building's elements to be exactly what the IFC puts
// in a spatial structure. That excludes openings, which are voids rather than
// things, and abstract entities.
func (c *converter) collectElements() {
	for _, r := range c.f.Of("IFCRELCONTAINEDINSPATIALSTRUCTURE") {
		for _, m := range step.List(r.Arg(4)) {
			if e := c.f.Get(m); e != nil {
				c.elements[e.ID] = e
			}
		}
	}
}

func (c *converter) elementsSection() {
	c.head("elements")
	unmapped := map[string]bool{}
	for _, e := range c.sorted() {
		classes := []string{"bot:Element"}
		pre := c.predefined(e)
		if m, ok := elementOntology[e.Type]; ok {
			ns, local := m[0], m[1]
			classes = append(classes, ns+":"+local)
			if pre != "" && pre != "NOTDEFINED" && pre != "USERDEFINED" {
				classes = append(classes, fmt.Sprintf("%s:%s-%s", ns, local, pre))
			}
		} else {
			unmapped[e.Type] = true
		}
		c.w("%s a %s ;", c.iri(e), strings.Join(classes, ", "))
		c.label(e, "    ")
	}

	if len(unmapped) > 0 {
		var names []string
		for n := range unmapped {
			names = append(names, n)
		}
		sort.Strings(names)
		c.w("")
		c.w("# No building or distribution element class for: %s.", strings.Join(names, ", "))
		c.w("# They are bot:Element only, rather than given a class that may not exist.")
	}

	c.w("")
	for _, r := range c.f.Of("IFCRELCONTAINEDINSPATIALSTRUCTURE") {
		host := c.f.Get(r.Arg(5))
		if host == nil {
			continue
		}
		for _, m := range step.List(r.Arg(4)) {
			if e := c.f.Get(m); e != nil {
				c.w("%s bot:containsElement %s .", c.iri(host), c.iri(e))
			}
		}
	}
}

// hosting follows wall -> opening -> window to say that the wall hosts the
// window, which is what BOT means by hostsElement.
func (c *converter) hosting() {
	filler := map[int]*step.Entity{} // opening -> what fills it
	for _, r := range c.f.Of("IFCRELFILLSELEMENT") {
		if o, e := c.f.Get(r.Arg(4)), c.f.Get(r.Arg(5)); o != nil && e != nil {
			filler[o.ID] = e
		}
	}
	if len(filler) == 0 {
		return
	}
	c.head("what hosts what")
	for _, r := range c.f.Of("IFCRELVOIDSELEMENT") {
		host, opening := c.f.Get(r.Arg(4)), c.f.Get(r.Arg(5))
		if host == nil || opening == nil {
			continue
		}
		if f, ok := filler[opening.ID]; ok {
			c.w("%s bot:hostsElement %s .", c.iri(host), c.iri(f))
		}
	}
}

// boundaries emits both the shortcut edge, which is what queries want, and the
// reified bot:Interface, which is where the inside/outside flag can live.
func (c *converter) boundaries() {
	rels := c.f.Of("IFCRELSPACEBOUNDARY")
	if len(rels) == 0 {
		return
	}
	c.head("what bounds what")
	for i, r := range rels {
		space, elem := c.f.Get(r.Arg(4)), c.f.Get(r.Arg(5))
		if space == nil || elem == nil {
			continue
		}
		c.w("%s bot:adjacentElement %s .", c.iri(space), c.iri(elem))
		iface := fmt.Sprintf("inst:interface-%d", i+1)
		c.w("%s a bot:Interface ; bot:interfaceOf %s, %s ; cot:boundaryFacing %s .",
			iface, c.iri(space), c.iri(elem), lit(step.Enum(r.Arg(8))))
	}
}

// types is the individual-and-definition split: an installed element realises a
// product, of which there is one description however many are installed.
func (c *converter) types() {
	rels := c.f.Of("IFCRELDEFINESBYTYPE")
	if len(rels) == 0 {
		return
	}
	c.head("products, and the elements that realise them")
	for _, r := range rels {
		t := c.f.Get(r.Arg(5))
		if t == nil {
			continue
		}
		c.w("%s a bpo:Product, bpo:Element ;", c.iri(t))
		c.label(t, "    ")
		for _, m := range step.List(r.Arg(4)) {
			if e := c.f.Get(m); e != nil {
				c.w("%s a bpo:SingularEntity ; bpo:realisesObject %s .", c.iri(e), c.iri(t))
			}
		}
	}
}

func (c *converter) systems() {
	members := map[int][]*step.Entity{}
	for _, r := range c.f.Of("IFCRELASSIGNSTOGROUP") {
		g := c.f.Get(r.Arg(6))
		if g == nil {
			continue
		}
		for _, m := range step.List(r.Arg(4)) {
			if e := c.f.Get(m); e != nil {
				members[g.ID] = append(members[g.ID], e)
			}
		}
	}
	if len(members) == 0 {
		return
	}

	c.head("flow systems and software systems")
	for _, g := range c.f.Of("IFCDISTRIBUTIONSYSTEM") {
		c.w("%s a fso:DistributionSystem ;", c.iri(g))
		c.label(g, "    ")
		for _, e := range members[g.ID] {
			c.w("%s fso:hasComponent %s .", c.iri(g), c.iri(e))
			c.w("%s a fso:%s .", c.iri(e), fsoRole(e.Type, c.predefined(e)))
		}
	}
	// The plain IfcSystems are the mbaigo ones: software, not fluid.
	for _, g := range c.f.Of("IFCSYSTEM") {
		c.w("%s a cot:SoftwareSystem ;", c.iri(g))
		c.label(g, "    ")
		for _, e := range members[g.ID] {
			c.w("%s cot:manages %s .", c.iri(g), c.iri(e))
		}
	}
}

// classification becomes a SKOS scheme, which is the usual way to carry a
// classification table in RDF.
func (c *converter) classification() {
	schemes := c.f.Of("IFCCLASSIFICATION")
	if len(schemes) == 0 {
		return
	}
	c.head("classification")
	for _, s := range schemes {
		c.w("%s a skos:ConceptScheme ;", c.iri(s))
		if n := step.Str(s.Arg(3)); n != "" {
			c.w("    skos:prefLabel %s ;", lit(n))
		}
		if d := step.Str(s.Arg(4)); d != "" {
			c.w("    rdfs:comment %s ;", lit(d))
		}
		c.w("    dcterms:source %s .", lit(step.Str(s.Arg(0))))
	}
	for i, r := range c.f.Of("IFCRELASSOCIATESCLASSIFICATION") {
		ref := c.f.Get(r.Arg(5))
		if ref == nil {
			continue
		}
		concept := fmt.Sprintf("inst:concept-%d", i+1)
		c.w("%s a skos:Concept ; skos:notation %s ; skos:prefLabel %s",
			concept, lit(step.Str(ref.Arg(1))), lit(step.Str(ref.Arg(2))))
		if scheme := c.f.Get(ref.Arg(3)); scheme != nil {
			c.w("    ; skos:inScheme %s", c.iri(scheme))
		}
		c.w("    .")
		for _, m := range step.List(r.Arg(4)) {
			if e := c.f.Get(m); e != nil {
				c.w("%s dcterms:subject %s .", c.iri(e), concept)
			}
		}
	}
}

func (c *converter) properties() {
	rels := c.f.Of("IFCRELDEFINESBYPROPERTIES")
	if len(rels) == 0 {
		return
	}
	c.head("property sets")
	for _, r := range rels {
		set := c.f.Get(r.Arg(5))
		if set == nil || set.Type != "IFCPROPERTYSET" {
			continue
		}
		for _, m := range step.List(r.Arg(4)) {
			subject := c.f.Get(m)
			if subject == nil {
				continue
			}
			for _, pr := range step.List(set.Arg(4)) {
				p := c.f.Get(pr)
				if p == nil || p.Type != "IFCPROPERTYSINGLEVALUE" {
					continue
				}
				name := step.Str(p.Arg(0))
				value, ok := literal(p.Arg(2))
				if name == "" || !ok {
					continue
				}
				c.w("%s cot:%s %s .", c.iri(subject), lowerCamel(name), value)
			}
		}
	}
}

// ---------------------------------------------------------------- helpers

func (c *converter) sorted() []*step.Entity {
	out := make([]*step.Entity, 0, len(c.elements))
	for _, e := range c.f.Order {
		if _, ok := c.elements[e.ID]; ok {
			out = append(out, e)
		}
	}
	return out
}

func (c *converter) predefined(e *step.Entity) string {
	at, ok := predefinedAt[e.Type]
	if !ok {
		at = 8
	}
	return step.Enum(e.Arg(at))
}

func (c *converter) label(e *step.Entity, indent string) {
	if n := step.Str(e.Arg(2)); n != "" {
		c.w("%srdfs:label %s ;", indent, lit(n))
	}
	c.w("%sdcterms:identifier %s .", indent, lit(step.Str(e.Arg(0))))
}

// iri names an entity by its IfcGloballyUniqueId, which is stable across
// regeneration. The $ that IFC's base-64 alphabet allows is percent encoded,
// since a Turtle prefixed name cannot hold it.
func (c *converter) iri(e *step.Entity) string {
	if g := step.Str(e.Arg(0)); len(g) == 22 {
		return "inst:" + strings.ReplaceAll(g, "$", "%24")
	}
	return fmt.Sprintf("inst:e%d", e.ID)
}

func lit(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`, "\t", `\t`)
	return `"` + r.Replace(s) + `"`
}

// literal turns an IFC nominal value such as IFCLABEL('x') or
// IFCPOWERMEASURE(2000.) into a Turtle literal.
func literal(arg string) (string, bool) {
	_, inner := step.Typed(arg)
	if s := step.Str(inner); s != "" {
		return lit(s), true
	}
	if v, ok := step.Num(inner); ok {
		return fmt.Sprintf(`"%g"^^xsd:double`, v), true
	}
	if e := step.Enum(inner); e != "" {
		return lit(e), true
	}
	return "", false
}

func lowerCamel(s string) string {
	var out []rune
	upper := false
	for i, r := range s {
		switch {
		case r == ' ' || r == '_' || r == '-':
			upper = true
		case i == 0:
			out = append(out, []rune(strings.ToLower(string(r)))[0])
		case upper:
			out = append(out, []rune(strings.ToUpper(string(r)))[0])
			upper = false
		default:
			out = append(out, r)
		}
	}
	return string(out)
}
