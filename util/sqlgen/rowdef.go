package sqlgen

type (
	// A RowDef serves to define individual rows within a FieldSet
	RowDef interface {
		// FD is short for Field Definition - used for defining fields in the row.
		FD(fmt string, col string, vals ...interface{})

		// CF is short for Candidate Field - used to define fields in the row that participate in distinguishing the row.
		// Two rows with the same values for their "CFs" will be considered the same.
		CF(fmt string, col string, vals ...interface{})

		KV(col string, val interface{})
	}

	// FieldDefFunc represents RowDef.FD and RowDef.CF.
	FieldDefFunc func(fmt, col string, vals ...interface{})

	rowdef struct {
		row      *row
		fieldset *fieldset
	}
)

func (r rowdef) deffield(fmt string, col string, vals []interface{}, cand bool) {
	column := r.fieldset.getcol(col, fmt, cand)
	(*r.row)[col] = field{column: column, values: vals}
}

// CF is short for Candidate Field - used to define fields in the row that partipate in distinguishing the row.
// Two rows with the same values for their "CFs" will be considered the same.
func (r rowdef) CF(fmt string, col string, vals ...interface{}) {
	r.deffield(fmt, col, vals, true)
}

// FD is short for Field Definition - used for defining fields in the row.
func (r rowdef) FD(fmt string, col string, vals ...interface{}) {
	r.deffield(fmt, col, vals, false)
}

// KV is short for Key-Value - used for defining simple fields in the row, where the value should be interpolated directly.
func (r rowdef) KV(col string, val interface{}) {
	r.deffield("?", col, []interface{}{val}, false)
}
