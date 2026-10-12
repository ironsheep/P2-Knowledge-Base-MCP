package obex

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ironsheep/p2kb-mcp/internal/kbtest"
	"gopkg.in/yaml.v3"
)

// categoryDoc is the shape the object parser uses, reduced to the category.
type categoryDoc struct {
	Category Category `yaml:"category"`
}

func parseCategory(t *testing.T, doc string) (Category, error) {
	t.Helper()
	var d categoryDoc
	err := yaml.Unmarshal([]byte(doc), &d)
	return d.Category, err
}

func TestCategoryParsesStringAndList(t *testing.T) {
	s, err := parseCategory(t, "category: Protocol\n")
	if err != nil {
		t.Fatalf("string category: %v", err)
	}
	if got := s.Values(); !reflect.DeepEqual(got, []string{"Protocol"}) || s.list {
		t.Errorf("string category = %v list=%v", got, s.list)
	}
	l, err := parseCategory(t, "category: [Math, Protocol]\n")
	if err != nil {
		t.Fatalf("list category: %v", err)
	}
	if got := l.Values(); !reflect.DeepEqual(got, []string{"Math", "Protocol"}) || !l.list {
		t.Errorf("list category = %v list=%v", got, l.list)
	}
}

func TestCategoryHasIgnoresCase(t *testing.T) {
	l := NewCategoryList("Math", "Protocol")
	for _, name := range []string{"protocol", "PROTOCOL", "Math"} {
		if !l.Has(name) {
			t.Errorf("list Has(%q) = false", name)
		}
	}
	if l.Has("Motor") || l.Has("") {
		t.Error("Has matched a name that is not there")
	}
	if !NewCategoryString("Protocol").Has("protocol") {
		t.Error("string Has(protocol) = false")
	}
	if (Category{}).Has("") || (Category{}).Has("x") {
		t.Error("empty category Has matched")
	}
}

func TestCategoryEmptyAndAbsent(t *testing.T) {
	for name, doc := range map[string]string{
		"absent":       "other: 1\n",
		"null":         "category:\n",
		"empty string": "category: ''\n",
		"empty list":   "category: []\n",
	} {
		c, err := parseCategory(t, doc)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		out, _ := json.Marshal(c)
		if len(c.values) != 0 || c.Values() != nil || c.String() != "" || string(out) != `""` {
			t.Errorf("%s: values=%v string=%q json=%s, want empty and \"\"", name, c.Values(), c.String(), out)
		}
	}
}

func TestCategoryJSONKeepsGivenForm(t *testing.T) {
	cases := []struct {
		name string
		c    Category
		want string
	}{
		{"string", NewCategoryString("Protocol"), `"Protocol"`},
		{"list", NewCategoryList("Math", "Protocol"), `["Math","Protocol"]`},
		{"one-element list stays a list", NewCategoryList("Protocol"), `["Protocol"]`},
		{"zero value", Category{}, `""`},
	}
	for _, tc := range cases {
		got, err := json.Marshal(tc.c)
		if err != nil || string(got) != tc.want {
			t.Errorf("%s: json = %s, %v; want %s", tc.name, got, err, tc.want)
		}
	}
	// Through the YAML path, a one-element list stays a list.
	c, err := parseCategory(t, "category: [Protocol]\n")
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := json.Marshal(c); string(got) != `["Protocol"]` {
		t.Errorf("parsed one-element list json = %s", got)
	}
	// And inside a struct, as SearchResult will carry it.
	got, _ := json.Marshal(struct {
		Category Category `json:"category"`
	}{NewCategoryList("A", "B")})
	if string(got) != `{"category":["A","B"]}` {
		t.Errorf("embedded json = %s", got)
	}
}

func TestCategoryRejectsOtherShapes(t *testing.T) {
	for name, doc := range map[string]string{
		"mapping":                "category:\n  name: Protocol\n",
		"list of mappings":       "category:\n- name: Protocol\n",
		"list of ints":           "category: [1, 2]\n",
		"list with nested list":  "category: [[a], b]\n",
		"list with null element": "category: [a, ~]\n",
		"integer scalar":         "category: 7\n",
	} {
		if _, err := parseCategory(t, doc); err == nil {
			t.Errorf("%s: parsed without error", name)
		}
	}
	// The same parser accepts the valid forms, so the errors above are the shape's doing.
	for _, doc := range []string{"category: Protocol\n", "category: [a, b]\n"} {
		if _, err := parseCategory(t, doc); err != nil {
			t.Errorf("%q: %v", doc, err)
		}
	}
}

// TestListCategoryFixtureFailsOnStringType shows the defect: the live 2811 body
// does not unmarshal into a plain-string category (the 1.5.0 type) and does
// into Category.
func TestListCategoryFixtureFailsOnStringType(t *testing.T) {
	body := []byte(kbtest.OBEXObject(kbtest.OBEXListCategoryID))

	var old struct {
		ObjectMetadata struct {
			Functionality struct {
				Category string `yaml:"category"`
			} `yaml:"functionality"`
		} `yaml:"object_metadata"`
	}
	if err := yaml.Unmarshal(body, &old); err == nil {
		t.Error("2811 unmarshalled into a string category; the fixture does not reproduce the defect")
	}

	var cur struct {
		ObjectMetadata struct {
			Functionality categoryDoc `yaml:"functionality"`
		} `yaml:"object_metadata"`
	}
	if err := yaml.Unmarshal(body, &cur); err != nil {
		t.Fatalf("2811 into Category: %v", err)
	}
	want := []string{"Math", "Motor Control", "Protocol", "Snippet"}
	if got := cur.ObjectMetadata.Functionality.Category.Values(); !reflect.DeepEqual(got, want) {
		t.Errorf("2811 category = %v, want %v", got, want)
	}
}

func TestMalformedFixtureCategoryFailsToParse(t *testing.T) {
	var cur struct {
		ObjectMetadata struct {
			Functionality categoryDoc `yaml:"functionality"`
		} `yaml:"object_metadata"`
	}
	if err := yaml.Unmarshal([]byte(kbtest.OBEXObject(kbtest.OBEXMalformedID)), &cur); err == nil {
		t.Error("malformed fixture parsed without error")
	}
}
