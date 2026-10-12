package obex

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Category is an OBEX object's functionality.category as the KB publishes it:
// a single string ("Protocol") or a list of strings ([Math, Protocol]). It
// remembers which, so JSON output keeps the form it was given.
//
// The zero value is the empty category: no values, emitted as "".
type Category struct {
	values []string
	list   bool
}

// NewCategoryString returns a string-form Category. An empty s is the empty category.
func NewCategoryString(s string) Category {
	if s == "" {
		return Category{}
	}
	return Category{values: []string{s}}
}

// NewCategoryList returns a list-form Category holding names, even for one name.
// No names is the empty category.
func NewCategoryList(names ...string) Category {
	if len(names) == 0 {
		return Category{}
	}
	return Category{values: append([]string(nil), names...), list: true}
}

// UnmarshalYAML accepts an absent/null/empty value (empty category), a string
// scalar, or a sequence of string scalars. Anything else (a mapping, a
// non-string scalar, a sequence holding a non-string) is an error.
func (c *Category) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Tag == "!!null" {
			*c = Category{}
			return nil
		}
		if node.Tag != "!!str" {
			return fmt.Errorf("line %d: category must be a string or a list of strings, got %s", node.Line, node.Tag)
		}
		*c = NewCategoryString(node.Value)
		return nil
	case yaml.SequenceNode:
		names := make([]string, 0, len(node.Content))
		for _, item := range node.Content {
			if item.Kind != yaml.ScalarNode || item.Tag != "!!str" {
				return fmt.Errorf("line %d: category list must hold only strings", item.Line)
			}
			names = append(names, item.Value)
		}
		*c = NewCategoryList(names...)
		return nil
	default:
		return fmt.Errorf("line %d: category must be a string or a list of strings, got a mapping", node.Line)
	}
}

// MarshalJSON emits a list-form Category as an array and anything else as a
// string ("" when empty).
func (c Category) MarshalJSON() ([]byte, error) {
	if c.list {
		return json.Marshal(c.values)
	}
	return json.Marshal(c.String())
}

// Values returns the category names, a copy; nil when empty.
func (c Category) Values() []string {
	if len(c.values) == 0 {
		return nil
	}
	return append([]string(nil), c.values...)
}

// Has reports whether name is one of the category names, ignoring case.
func (c Category) Has(name string) bool {
	if name == "" {
		return false
	}
	for _, v := range c.values {
		if strings.EqualFold(v, name) {
			return true
		}
	}
	return false
}

// String joins the names with ", " for display; "" when empty.
func (c Category) String() string { return strings.Join(c.values, ", ") }
