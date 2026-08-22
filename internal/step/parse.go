// Package step reads STEP physical files (ISO 10303-21), which is the format
// IFC is normally exchanged in. It understands the file's shape, not the IFC
// schema: entities come back as a type name and a list of unparsed arguments,
// and it is up to the caller to know what those arguments mean.
package step

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Entity is one instance line: #12=IFCWALL(...).
type Entity struct {
	ID   int
	Type string
	Args []string
}

// File is a parsed STEP file, indexed by instance number and by type.
type File struct {
	ByID   map[int]*Entity
	byType map[string][]*Entity
	Order  []*Entity // as they appeared, so output is deterministic
}

// Of returns every entity of exactly this type, in file order. Subtypes are not
// included: IfcDistributionSystem does not come back under IfcSystem.
func (f *File) Of(typ string) []*Entity { return f.byType[strings.ToUpper(typ)] }

// Get resolves a reference argument such as "#42".
func (f *File) Get(arg string) *Entity {
	if id := Ref(arg); id != 0 {
		return f.ByID[id]
	}
	return nil
}

// Arg returns the i'th argument of an entity, or "" if it has fewer.
func (e *Entity) Arg(i int) string {
	if e == nil || i >= len(e.Args) {
		return ""
	}
	return e.Args[i]
}

// Ref is the instance number of a reference argument, or 0.
func Ref(arg string) int {
	if len(arg) > 1 && arg[0] == '#' {
		n, err := strconv.Atoi(arg[1:])
		if err == nil {
			return n
		}
	}
	return 0
}

// Str unquotes a string argument, undoing STEP's doubled-quote escape.
func Str(arg string) string {
	if len(arg) >= 2 && arg[0] == '\'' && arg[len(arg)-1] == '\'' {
		return strings.ReplaceAll(arg[1:len(arg)-1], "''", "'")
	}
	return ""
}

// Enum strips the dots from .SOLIDWALL. and returns SOLIDWALL.
func Enum(arg string) string {
	if len(arg) >= 2 && arg[0] == '.' && arg[len(arg)-1] == '.' {
		return arg[1 : len(arg)-1]
	}
	return ""
}

// Num parses a numeric argument.
func Num(arg string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSuffix(arg, "."), 64)
	return v, err == nil
}

// List splits an aggregate argument "(#1,#2)" into its members.
func List(arg string) []string {
	if len(arg) >= 2 && arg[0] == '(' && arg[len(arg)-1] == ')' {
		return SplitArgs(arg[1 : len(arg)-1])
	}
	return nil
}

// Typed unwraps a defined type such as IFCLABEL('x'), returning the type name
// and the inner argument.
func Typed(arg string) (string, string) {
	open := strings.IndexByte(arg, '(')
	if open <= 0 || !strings.HasSuffix(arg, ")") {
		return "", arg
	}
	name := strings.TrimSpace(arg[:open])
	for _, r := range name {
		if !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && r != '_' {
			return "", arg
		}
	}
	return name, arg[open+1 : len(arg)-1]
}

// SplitArgs splits a comma separated argument list, respecting nesting and
// quoted strings.
func SplitArgs(s string) []string {
	var out []string
	depth, inStr := 0, false
	var cur strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			cur.WriteByte(c)
			if c == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' {
					i++
					cur.WriteByte('\'')
				} else {
					inStr = false
				}
			}
			continue
		}
		switch c {
		case '\'':
			inStr = true
			cur.WriteByte(c)
		case '(':
			depth++
			cur.WriteByte(c)
		case ')':
			depth--
			cur.WriteByte(c)
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(cur.String()))
				cur.Reset()
				continue
			}
			cur.WriteByte(c)
		default:
			cur.WriteByte(c)
		}
	}
	out = append(out, strings.TrimSpace(cur.String()))
	return out
}

// Parse reads a STEP file. Instances may span lines, so the data section is
// split on semicolons outside strings rather than read line by line.
func Parse(r io.Reader) (*File, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	text := string(raw)

	start := strings.Index(text, "DATA;")
	if start < 0 {
		return nil, fmt.Errorf("step: no DATA section")
	}
	body := text[start+len("DATA;"):]
	if end := strings.Index(body, "ENDSEC;"); end >= 0 {
		body = body[:end]
	}

	f := &File{ByID: map[int]*Entity{}, byType: map[string][]*Entity{}}
	for _, stmt := range statements(body) {
		e, ok := parseInstance(stmt)
		if !ok {
			continue
		}
		f.ByID[e.ID] = e
		f.byType[e.Type] = append(f.byType[e.Type], e)
		f.Order = append(f.Order, e)
	}
	if len(f.Order) == 0 {
		return nil, fmt.Errorf("step: data section held no instances")
	}
	return f, nil
}

func statements(body string) []string {
	var out []string
	inStr := false
	var cur strings.Builder
	for i := 0; i < len(body); i++ {
		c := body[i]
		if inStr {
			cur.WriteByte(c)
			if c == '\'' {
				if i+1 < len(body) && body[i+1] == '\'' {
					i++
					cur.WriteByte('\'')
				} else {
					inStr = false
				}
			}
			continue
		}
		if c == '\'' {
			inStr = true
			cur.WriteByte(c)
			continue
		}
		if c == ';' {
			out = append(out, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	return out
}

func parseInstance(stmt string) (*Entity, bool) {
	s := strings.TrimSpace(stmt)
	if len(s) == 0 || s[0] != '#' {
		return nil, false
	}
	eq := strings.IndexByte(s, '=')
	if eq < 0 {
		return nil, false
	}
	id, err := strconv.Atoi(strings.TrimSpace(s[1:eq]))
	if err != nil {
		return nil, false
	}
	rest := strings.TrimSpace(s[eq+1:])
	open := strings.IndexByte(rest, '(')
	if open < 0 || !strings.HasSuffix(rest, ")") {
		return nil, false
	}
	return &Entity{
		ID:   id,
		Type: strings.ToUpper(strings.TrimSpace(rest[:open])),
		Args: SplitArgs(rest[open+1 : len(rest)-1]),
	}, true
}
