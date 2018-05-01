package sqlgen

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strings"
)

type (
	// A FieldSet collects the requirements of a set of fields,
	// and then helps generate SQL related to it.
	FieldSet interface {
		// It adds a row to the set.
		Row(func(RowDef))

		// Potent returns true if this FieldSet would produce a useful SQL query.
		// If Potent() returns false, there's no purpose in using this FieldSet with the database.
		Potent() bool

		// InsertSQL generates and returns an INSERT query to send to the database to insert this FieldSet.
		// The conflict string is used to resolve duplicate rows in an upserty kind of a way.
		// It allows Go template formatting.
		InsertSQL(table string, conflict string) string

		// InsertValues produces a slice of values for INSERTs, such that the positions of values will match up
		// with SQL replacements in an InsertSQL call.
		InsertValues() []interface{}

		// RowCount returns the number of rows in the FieldSet.
		RowCount() int
	}

	fieldset struct {
		colnames []string
		coldefs  map[string]*coldef
		rows     []row
	}
)

// NewFieldset returns a new FieldSet
func NewFieldset() FieldSet {
	return &fieldset{
		coldefs: map[string]*coldef{},
		rows:    []row{},
	}
}

// Row implements FieldSet on fieldset. It adds a row to the set.
func (f *fieldset) Row(fn func(RowDef)) {
	row := row{}
	def := rowdef{row: &row, fieldset: f}
	fn(def)
	for _, r := range f.rows {
		if r.dupes(row) {
			return //kick back dupes
		}
	}
	f.rows = append(f.rows, row)
}

// Potent implents FieldSet.
func (f fieldset) Potent() bool {
	return len(f.colnames) > 0
}

// InsertSQL implements FieldSet on fieldset.
func (f fieldset) InsertSQL(table, conflict string) string {
	vs := f.values()
	return fmt.Sprintf("insert into %s %s values %s %s", table, f.columns(), vs, f.conflictClause(conflict))
}

// InsertValues implements FieldSet on fieldset.
func (f fieldset) InsertValues() []interface{} {
	vals := []interface{}{}
	for _, r := range f.rows {
		for _, name := range f.colnames {
			vals = append(vals, r[name].values...)
		}
	}
	return vals
}

// RowCount implements FieldSet on fieldset.
func (f fieldset) RowCount() int {
	return len(f.rows)
}

func (f *fieldset) getcol(col, frmt string, cand bool) *coldef {
	if c, has := f.coldefs[col]; has {
		if col != c.name || frmt != c.fmt || cand != c.candidate {
			panic(fmt.Sprintf("Mismatched coldef: %#v != %q %q", c, col, frmt))
		}
		return c
	}
	c := &coldef{name: col, fmt: frmt, candidate: cand}
	f.coldefs[col] = c
	f.colnames = append(f.colnames, col)
	return c
}

func (f fieldset) conflictClause(templ string) string {
	buf := &bytes.Buffer{}
	conflictTemplate := template.Must(template.New("conflict").Parse(templ))
	conflictTemplate.Execute(buf, f)
	return buf.String()
}

func (f fieldset) columns() string {
	return "(" + strings.Join(f.quotedColnames(), ",") + ")"
}

func (f fieldset) quotedColnames() []string {
	qcns := []string{}
	for _, cn := range f.colnames {
		qcns = append(qcns, `"`+cn+`"`)
	}
	return qcns
}

// Candidates returns the index candidate columns for this fieldset.
func (f fieldset) Candidates() string {
	return f.candidates()
}

func (f fieldset) candidates() string {
	colnames := []string{}
	for _, name := range f.colnames {
		if f.coldefs[name].candidate {
			colnames = append(colnames, name)
		}
	}
	return "(" + strings.Join(colnames, ",") + ")"
}

// NonCandidates returns noncandidate column names for this fieldset.
func (f fieldset) NonCandidates() string {
	return f.noncandidates()
}

// NSNonCandidates returns noncandidate columns namespaced with a table name.
func (f fieldset) NSNonCandidates(namespace string) string {
	colnames := []string{}
	for _, name := range f.colnames {
		if !f.coldefs[name].candidate {
			colnames = append(colnames, namespace+"."+name)
		}
	}
	return "(" + strings.Join(colnames, ",") + ")"
}

func (f fieldset) noncandidates() string {
	colnames := []string{}
	for _, name := range f.colnames {
		if !f.coldefs[name].candidate {
			colnames = append(colnames, name)
		}
	}
	return "(" + strings.Join(colnames, ",") + ")"
}

var placeholderQs = regexp.MustCompile(`\?`)

func (f fieldset) values() string {
	placeIdx := 0

	lines := []string{}
	for range f.rows {
		valpats := []string{}
		for _, name := range f.colnames {
			pat := f.coldefs[name].fmt
			pat = placeholderQs.ReplaceAllStringFunc(pat, func(q string) string {
				placeIdx++
				return fmt.Sprintf("$%d", placeIdx)
			})

			valpats = append(valpats, pat)
		}
		format := "(" + strings.Join(valpats, ",") + ")"
		lines = append(lines, format)
	}

	return strings.Join(lines, ",\n")
}
