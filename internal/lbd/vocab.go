package lbd

// The ontology terms used here were checked against the published ontologies
// rather than recalled: BOT and OMG/FOG from w3id.org, the Building Element and
// Distribution Element ontologies from pi.pauwel.be, FSO and BPO from w3id.org.
// Every class this file can emit exists in the version fetched.
const (
	nsBOT     = "https://w3id.org/bot#"
	nsBEO     = "https://pi.pauwel.be/voc/buildingelement#"
	nsMEP     = "https://pi.pauwel.be/voc/distributionelement#"
	nsFSO     = "https://w3id.org/fso#"
	nsBPO     = "https://w3id.org/bpo#"
	nsOMG     = "https://w3id.org/omg#"
	nsFOG     = "https://w3id.org/fog#"
	nsSKOS    = "http://www.w3.org/2004/02/skos/core#"
	nsDCTERMS = "http://purl.org/dc/terms/"
	nsRDFS    = "http://www.w3.org/2000/01/rdf-schema#"
	nsOWL     = "http://www.w3.org/2002/07/owl#"
	nsXSD     = "http://www.w3.org/2001/XMLSchema#"
)

// elementOntology maps an IFC entity onto the ontology and class that describe
// it. The local names have to be spelled out because IFC files write entity
// names in capitals, so CamelCase cannot be recovered from the file. An entity
// that is not listed still becomes a bot:Element; it just gets no more specific
// class, which is better than guessing at one that may not exist.
var elementOntology = map[string][2]string{
	// Building elements.
	"IFCWALL": {"beo", "Wall"},
	// BEO has no WallStandardCase: the standard case is a predefined type of Wall.
	"IFCWALLSTANDARDCASE": {"beo", "Wall"},
	"IFCCURTAINWALL":      {"beo", "CurtainWall"},
	"IFCWINDOW":           {"beo", "Window"},
	"IFCDOOR":             {"beo", "Door"},
	"IFCSLAB":             {"beo", "Slab"},
	"IFCROOF":             {"beo", "Roof"},
	"IFCBEAM":             {"beo", "Beam"},
	"IFCCOLUMN":           {"beo", "Column"},
	"IFCCOVERING":         {"beo", "Covering"},
	"IFCSTAIR":            {"beo", "Stair"},
	"IFCRAILING":          {"beo", "Railing"},
	"IFCPLATE":            {"beo", "Plate"},
	"IFCMEMBER":           {"beo", "Member"},
	"IFCFOOTING":          {"beo", "Footing"},
	"IFCPILE":             {"beo", "Pile"},
	"IFCRAMP":             {"beo", "Ramp"},
	"IFCCHIMNEY":          {"beo", "Chimney"},
	"IFCSHADINGDEVICE":    {"beo", "ShadingDevice"},

	// Distribution elements.
	"IFCSPACEHEATER":                {"mep", "SpaceHeater"},
	"IFCSENSOR":                     {"mep", "Sensor"},
	"IFCOUTLET":                     {"mep", "Outlet"},
	"IFCSANITARYTERMINAL":           {"mep", "SanitaryTerminal"},
	"IFCELECTRICAPPLIANCE":          {"mep", "ElectricAppliance"},
	"IFCPIPESEGMENT":                {"mep", "PipeSegment"},
	"IFCDUCTSEGMENT":                {"mep", "DuctSegment"},
	"IFCPIPEFITTING":                {"mep", "PipeFitting"},
	"IFCDUCTFITTING":                {"mep", "DuctFitting"},
	"IFCDISTRIBUTIONCHAMBERELEMENT": {"mep", "DistributionChamberElement"},
	"IFCVALVE":                      {"mep", "Valve"},
	"IFCPUMP":                       {"mep", "Pump"},
	"IFCFAN":                        {"mep", "Fan"},
	"IFCDAMPER":                     {"mep", "Damper"},
	"IFCBOILER":                     {"mep", "Boiler"},
	"IFCTANK":                       {"mep", "Tank"},
	"IFCAIRTERMINAL":                {"mep", "AirTerminal"},
	"IFCLIGHTFIXTURE":               {"mep", "LightFixture"},
	"IFCSWITCHINGDEVICE":            {"mep", "SwitchingDevice"},
	"IFCFLOWMETER":                  {"mep", "FlowMeter"},
}

// predefinedAt is where an entity keeps its PredefinedType. Most elements put
// it straight after Tag; windows and doors carry two size attributes first.
var predefinedAt = map[string]int{
	"IFCWINDOW": 10, "IFCDOOR": 10,
}

// fsoRole places a component in a flow system. Only the clear cases are
// mapped; anything else is left as a plain fso:Component.
func fsoRole(entity, predefined string) string {
	switch entity {
	case "IFCPIPESEGMENT", "IFCDUCTSEGMENT":
		return "Segment"
	case "IFCPIPEFITTING", "IFCDUCTFITTING":
		return "Fitting"
	case "IFCSANITARYTERMINAL", "IFCSPACEHEATER", "IFCAIRTERMINAL":
		return "Terminal"
	case "IFCVALVE", "IFCDAMPER":
		return "FlowController"
	case "IFCPUMP", "IFCFAN":
		return "FlowMovingDevice"
	case "IFCTANK":
		return "StorageDevice"
	case "IFCELECTRICAPPLIANCE":
		if predefined == "FREESTANDINGWATERHEATER" {
			return "StorageDevice"
		}
	}
	return "Component"
}
