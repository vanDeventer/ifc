// Package ifc writes IFC4 models in the STEP physical file format (ISO 10303-21).
//
// It is deliberately small: it knows how to number entity instances, format
// STEP values and hand out IFC globally unique identifiers. Knowledge of which
// entities make up a building lives in the packages that use it.
package ifc

import (
	"crypto/md5"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// Ref is a reference to an entity instance, written as #1, #2, ... The zero
// value means "no reference" and is written as the STEP null value $.
type Ref int

// Enum is a STEP enumeration value, written as .VALUE.
type Enum string

// Star is the STEP redeclared/derived value, written as *.
type Star struct{}

// Null is the STEP unset value, written as $.
type Null struct{}

// List is an ordered STEP aggregate, written as (a,b,c).
type List []any

// Typed is a value wrapped in its defined type, written as IFCLABEL('x'). IFC
// needs this wherever an attribute is a SELECT over measure types, as the
// nominal value of a property is.
type Typed struct {
	Type  string
	Value any
}

// Label, Text and Power are the typed values this package needs most.
func Label(s string) Typed  { return Typed{"IFCLABEL", s} }
func Text(s string) Typed   { return Typed{"IFCTEXT", s} }
func Power(w float64) Typed { return Typed{"IFCPOWERMEASURE", w} }

// File collects entity instances in the order they are added.
type File struct {
	Description  string
	Name         string
	Author       string
	Organization string
	Timestamp    time.Time

	lines []string
}

// New returns an empty file. The timestamp is written into the header and is
// the only source of time in the package, so the same inputs always produce a
// byte-identical file.
func New(name, author, org string, stamp time.Time) *File {
	return &File{
		Description:  "ViewDefinition [CoordinationView]",
		Name:         name,
		Author:       author,
		Organization: org,
		Timestamp:    stamp,
	}
}

// Add appends an entity instance and returns its reference.
func (f *File) Add(typ string, args ...any) Ref {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = value(a)
	}
	id := Ref(len(f.lines) + 1)
	f.lines = append(f.lines, fmt.Sprintf("#%d=%s(%s);", id, typ, strings.Join(parts, ",")))
	return id
}

// Write writes the complete STEP file.
func (f *File) Write(w io.Writer) error {
	stamp := f.Timestamp.UTC().Format("2006-01-02T15:04:05")
	var b strings.Builder
	b.WriteString("ISO-10303-21;\n")
	b.WriteString("HEADER;\n")
	fmt.Fprintf(&b, "FILE_DESCRIPTION((%s),'2;1');\n", value(f.Description))
	fmt.Fprintf(&b, "FILE_NAME(%s,%s,(%s),(%s),%s,%s,'');\n",
		value(f.Name), value(stamp), value(f.Author), value(f.Organization),
		value("ifcgen"), value("github.com/vanDeventer/ifc"))
	b.WriteString("FILE_SCHEMA(('IFC4'));\n")
	b.WriteString("ENDSEC;\n")
	b.WriteString("DATA;\n")
	for _, l := range f.lines {
		b.WriteString(l)
		b.WriteByte('\n')
	}
	b.WriteString("ENDSEC;\n")
	b.WriteString("END-ISO-10303-21;\n")
	_, err := io.WriteString(w, b.String())
	return err
}

// guidChars is the base-64 alphabet IFC uses for its compressed GUIDs. It is
// not RFC 4648: the last two characters are _ and $.
const guidChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_$"

// GUID derives a stable IfcGloballyUniqueId from seed, so that regenerating
// the model does not churn every identifier. The hash is an identifier, not a
// security primitive.
func GUID(seed string) string {
	sum := md5.Sum([]byte(seed))
	n := new(big.Int).SetBytes(sum[:])
	base := big.NewInt(64)
	rem := new(big.Int)
	out := make([]byte, 22)
	for i := 21; i >= 0; i-- {
		n.DivMod(n, base, rem)
		out[i] = guidChars[rem.Int64()]
	}
	return string(out)
}

func value(v any) string {
	switch t := v.(type) {
	case nil, Null:
		return "$"
	case Star:
		return "*"
	case Ref:
		if t == 0 {
			return "$"
		}
		return "#" + strconv.Itoa(int(t))
	case Enum:
		return "." + string(t) + "."
	case Typed:
		return t.Type + "(" + value(t.Value) + ")"
	case bool:
		if t {
			return ".T."
		}
		return ".F."
	case int:
		return strconv.Itoa(t)
	case float64:
		return stepReal(t)
	case string:
		return "'" + strings.ReplaceAll(t, "'", "''") + "'"
	case List:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = value(e)
		}
		return "(" + strings.Join(parts, ",") + ")"
	case []Ref:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = value(e)
		}
		return "(" + strings.Join(parts, ",") + ")"
	case []float64:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = stepReal(e)
		}
		return "(" + strings.Join(parts, ",") + ")"
	}
	panic(fmt.Sprintf("ifc: cannot write %T as a STEP value", v))
}

// stepReal formats a float the way STEP requires: always with a decimal point.
func stepReal(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += "."
	}
	return s
}
