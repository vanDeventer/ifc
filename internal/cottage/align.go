package cottage

import (
	"fmt"
	"strings"
)

// AlignOptions configures the alignment graph.
type AlignOptions struct {
	Base string // the namespace the IFC spaces were named in
	ALC  string // the local cloud's namespace, where mbaigo mints its values
}

// Alignment writes a third graph joining the mbaigo runtime graph to this
// building model.
//
// It is deliberately a separate file. It asserts things about a namespace this
// project does not own, and it is only true of one local cloud, so keeping it
// out of cottage.ttl leaves the building model good on its own.
//
// It is deliberately not owl:sameAs. Two of this cloud's functional locations,
// Kitchen and Entrance, are in the same open room. Written as identity,
//
//	alc:Kitchen  owl:sameAs inst:Living .
//	alc:Entrance owl:sameAs inst:Living .
//
// sameAs is symmetric and transitive, so a reasoner concludes alc:Kitchen
// owl:sameAs alc:Entrance and merges everything said about either. The relation
// is many functional locations to one room, which identity cannot express.
// cot:locatedIn says only what is true: the functional location is in that room.
func Alignment(p Params, opt AlignOptions) string {
	if opt.Base == "" {
		opt.Base = "https://example.org/building/"
	}
	if opt.ALC == "" {
		opt.ALC = "http://www.synecdoque.com/lcloud/"
	}
	voc := opt.Base + "vocab#"

	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }

	w("@prefix inst: <%s> .", opt.Base)
	w("@prefix cot:  <%s> .", voc)
	w("@prefix alc:  <%s> .", opt.ALC)
	w("@prefix owl:  <http://www.w3.org/2002/07/owl#> .")
	w("@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .")
	w("")
	w("# Joins the mbaigo runtime graph to the building model.")
	w("#")
	w("# Load this alongside cottage.ttl and the cloud's graph. It is kept out of")
	w("# both because it speaks about a namespace neither side owns and is true of")
	w("# one local cloud only.")
	w("#")
	w("# Not owl:sameAs: Kitchen and Entrance are both in the one open room here,")
	w("# and identity is symmetric and transitive, so asserting it twice would make")
	w("# the kitchen and the entrance the same thing. The relation is many to one.")
	w("")
	w("cot:locatedIn a owl:ObjectProperty ;")
	w("    rdfs:label %q ;", "located in")
	w("    rdfs:comment %q .", "Relates an mbaigo functional location to the IFC space it is in. "+
		"Many functional locations may share one space, so this is not identity.")
	w("")

	var placed int
	for _, s := range p.Spaces {
		if len(s.Functional) == 0 {
			continue
		}
		w("# %s", s.Long)
		for _, fl := range s.Functional {
			w("alc:%s cot:locatedIn inst:%s .", fl, escapeLocal(SpaceGUID(s.Name)))
			placed++
		}
		w("")
	}

	w("# %d functional location(s) placed in %d room(s).", placed, len(p.Spaces))
	w("# A functional location the cloud uses that is not listed here is either in")
	w("# another building or not a room at all: Outdoor, Lab and Plant are all three.")
	return b.String()
}

// escapeLocal makes an IfcGloballyUniqueId safe as a Turtle prefixed name. The
// identifier alphabet includes $, which PN_LOCAL does not allow.
func escapeLocal(guid string) string { return strings.ReplaceAll(guid, "$", "%24") }
