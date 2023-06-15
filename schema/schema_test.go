package schema

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSchema_FindTableByName(t *testing.T) {
	schema := Schema{
		Name: "testschema",
		Tables: []*Table{
			&Table{
				Name:    "a",
				Comment: "table a",
			},
			&Table{
				Name:    "b",
				Comment: "table b",
			},
		},
	}
	table, _ := schema.FindTableByName("b")
	want := "table b"
	got := table.Comment
	if got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestTable_FindColumnByName(t *testing.T) {
	table := Table{
		Name: "testtable",
		Columns: []*Column{
			&Column{
				Name:    "a",
				Comment: "column a",
			},
			&Column{
				Name:    "b",
				Comment: "column b",
			},
		},
	}
	column, _ := table.FindColumnByName("b")
	want := "column b"
	got := column.Comment
	if got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestTable_FindConstrainsByColumnName(t *testing.T) {
	table := Table{
		Name: "testtable",
		Columns: []*Column{
			&Column{
				Name:    "a",
				Comment: "column a",
			},
			&Column{
				Name:    "b",
				Comment: "column b",
			},
		},
	}
	table.Constraints = []*Constraint{
		&Constraint{
			Name:              "PRIMARY",
			Type:              "PRIMARY KEY",
			Def:               "PRIMARY KEY(a)",
			ReferencedTable:   nil,
			Table:             &table.Name,
			Columns:           []string{"a"},
			ReferencedColumns: []string{},
		},
		&Constraint{
			Name:              "UNIQUE",
			Type:              "UNIQUE",
			Def:               "UNIQUE KEY a (b)",
			ReferencedTable:   nil,
			Table:             &table.Name,
			Columns:           []string{"b"},
			ReferencedColumns: []string{},
		},
	}

	got := table.FindConstrainsByColumnName("a")
	if want := 1; len(got) != want {
		t.Errorf("got %v\nwant %v", len(got), want)
	}
	if want := "PRIMARY"; got[0].Name != want {
		t.Errorf("got %v\nwant %v", got[0].Name, want)
	}
}

func TestTable_hasColumnWithValues(t *testing.T) {
	tests := []struct {
		testName  string
		name      string
		addColumn *Column
		want      bool
	}{
		{"Has no ExtraDef value.", ColumnExtraDef, &Column{Name: "b"}, false},
		{"Has ExtraDef value", ColumnExtraDef, &Column{Name: "b", ExtraDef: "ExtraDef"}, true},
		{"Occurrences is invalid", ColumnOccurrences, &Column{Name: "b", Occurrences: sql.NullInt32{Valid: false}}, false},
		{"Occurrences is valid", ColumnOccurrences, &Column{Name: "b", Occurrences: sql.NullInt32{Valid: true}}, true},
		{"Percents is invalid", ColumnPercents, &Column{Name: "b", Percents: sql.NullFloat64{Valid: false}}, false},
		{"Percents is valid", ColumnPercents, &Column{Name: "b", Percents: sql.NullFloat64{Valid: true}}, true},
		{"Has no ChildRelations", ColumnChildren, &Column{Name: "b"}, false},
		{"Has ChildRelations", ColumnChildren, &Column{Name: "b", ChildRelations: []*Relation{{}}}, true},
		{"Has no ParentRelations", ColumnParents, &Column{Name: "b"}, false},
		{"Has ParentRelations", ColumnParents, &Column{Name: "b", ParentRelations: []*Relation{{}}}, true},
		{"Has no Comment", ColumnComment, &Column{Name: "b"}, false},
		{"Has Comment", ColumnComment, &Column{Name: "b", Comment: "comment"}, true},
		{"Has no Labels", ColumnLabels, &Column{Name: "b"}, false},
		{"Has Labels", ColumnLabels, &Column{Name: "b", Labels: Labels{{Name: "TestLabel"}}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			table := Table{
				Name: "testTable",
				Columns: []*Column{
					&Column{
						Name: "a",
					},
				},
			}
			table.Columns = append(table.Columns, tt.addColumn)

			got := table.hasColumnWithValues(tt.name)
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestTable_ShowColumn(t *testing.T) {
	tests := []struct {
		testName    string
		table       Table
		name        string
		hideColumns []string
		want        bool
	}{
		{"No hideColumns", Table{Name: "testTable"}, ColumnComment, []string{}, true},
		{"hideColumns without value", Table{Name: "testTable"}, ColumnComment, []string{ColumnComment}, false},
		{"hideColumns with value", Table{Name: "testTable", Columns: []*Column{{Name: "testColumn", Comment: "comment"}}}, ColumnComment, []string{ColumnComment}, true},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			got := tt.table.ShowColumn(tt.name, tt.hideColumns)
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestSchema_Sort(t *testing.T) {
	schema := Schema{
		Name: "testschema",
		Tables: []*Table{
			&Table{
				Name:    "b",
				Comment: "table b",
			},
			&Table{
				Name:    "a",
				Comment: "table a",
				Columns: []*Column{
					&Column{
						Name:    "b",
						Comment: "column b",
					},
					&Column{
						Name:    "a",
						Comment: "column a",
					},
				},
			},
		},
	}
	if err := schema.Sort(); err != nil {
		t.Error(err)
	}
	want := "a"
	got := schema.Tables[0].Name
	if got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
	want2 := "a"
	got2 := schema.Tables[0].Columns[0].Name
	if got2 != want2 {
		t.Errorf("got %v\nwant %v", got2, want2)
	}
}

func TestRepair(t *testing.T) {
	got := &Schema{}
	f, err := os.Open(filepath.Join(testdataDir(), "test_repair.golden"))
	if err != nil {
		t.Error(err)
	}
	dec := json.NewDecoder(f)
	if err := dec.Decode(got); err != nil {
		t.Error(err)
	}
	if err := got.Repair(); err != nil {
		t.Error(err)
	}

	want := newTestSchema(t)

	if diff := cmp.Diff(got, want, nil); diff != "" {
		t.Errorf("%s", diff)
	}

	b, err := json.Marshal(want)
	if err != nil {
		t.Error(err)
	}
	want2 := &Schema{}
	if err := json.Unmarshal(b, want2); err != nil {
		t.Error(err)
	}
	if err := want2.Repair(); err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(want, want2, nil); diff != "" {
		t.Errorf("%s", diff)
	}
}

func TestClone(t *testing.T) {
	want := newTestSchema(t)
	got, err := want.Clone()
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(got, want, nil); diff != "" {
		t.Errorf("%s", diff)
	}
}

func testdataDir() string {
	wd, _ := os.Getwd()
	dir, _ := filepath.Abs(filepath.Join(filepath.Dir(wd), "testdata"))
	return dir
}

func newTestSchema(t *testing.T) *Schema {
	ca := &Column{
		Name:     "a",
		Type:     "bigint(20)",
		Comment:  "column a",
		Nullable: false,
	}
	cb := &Column{
		Name:     "b",
		Type:     "text",
		Comment:  "column b",
		Nullable: true,
	}

	ta := &Table{
		Name:    "a",
		Type:    "BASE TABLE",
		Comment: "table a",
		Columns: []*Column{
			ca,
			&Column{
				Name:     "a2",
				Type:     "datetime",
				Comment:  "column a2",
				Nullable: false,
				Default: sql.NullString{
					String: "CURRENT_TIMESTAMP",
					Valid:  true,
				},
			},
		},
	}

	tb := &Table{
		Name:    "b",
		Type:    "BASE TABLE",
		Comment: "table b",
		Columns: []*Column{
			cb,
			&Column{
				Name:     "b2",
				Comment:  "column b2",
				Type:     "text",
				Nullable: true,
			},
		},
	}
	r := &Relation{
		Table:         ta,
		Columns:       []*Column{ca},
		ParentTable:   tb,
		ParentColumns: []*Column{cb},
	}
	ca.ParentRelations = []*Relation{r}
	cb.ChildRelations = []*Relation{r}

	s := &Schema{
		Name: "testschema",
		Tables: []*Table{
			ta,
			tb,
		},
		Relations: []*Relation{
			r,
		},
		Driver: &Driver{
			Name:            "testdriver",
			DatabaseVersion: "1.0.0",
			Meta:            &DriverMeta{},
		},
	}
	return s
}
