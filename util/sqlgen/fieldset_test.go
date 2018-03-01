package sqlgen

import "testing"

func TestRowDedupe(t *testing.T) {
	fs := NewFieldset()

	fs.Row(func(rd RowDef) { rd.CF("?", "testcol", 1) })
	fs.Row(func(rd RowDef) { rd.CF("?", "testcol", 1) })
	fs.Row(func(rd RowDef) { rd.CF("?", "testcol", 2) })

	sql := fs.InsertSQL("testtable", "")
	expectedSQL := "insert into testtable (testcol) values ($1),\n($2) "
	if sql != expectedSQL {
		t.Errorf("Expected fieldset to generate %q; got %q", expectedSQL, sql)
	}

	vals := fs.InsertValues()
	if len(vals) != 2 {
		t.Errorf("Expected 2 values, got %d", len(vals))
	}

	if vals[0] != 1 {
		t.Errorf("Expected first value to be 1, got %v", vals[0])
	}
	if vals[1] != 2 {
		t.Errorf("Expected first value to be 1, got %v", vals[1])
	}
}

func TestRowDedupe_without_candidates(t *testing.T) {
	fs := NewFieldset()

	fs.Row(func(rd RowDef) { rd.FD("?", "testcol", 1) })
	fs.Row(func(rd RowDef) { rd.FD("?", "testcol", 1) })
	fs.Row(func(rd RowDef) { rd.FD("?", "testcol", 2) })

	sql := fs.InsertSQL("testtable", "")
	expectedSQL := "insert into testtable (testcol) values ($1),\n($2),\n($3) "
	if sql != expectedSQL {
		t.Errorf("Expected fieldset to generate %q; got %q", expectedSQL, sql)
	}

	vals := fs.InsertValues()
	if len(vals) != 3 {
		t.Errorf("Expected 3 values, got %d", len(vals))
	}
}
