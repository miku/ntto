package ntto

import (
	"errors"
	"reflect"
	"testing"
)

var IsURIRefTests = []struct {
	in  string
	out bool
}{
	{"<http://example.org/show/218>", true},
	{"<http://example.org/elements/atomicNumber>", true},
	{"<http://example.org/elements/atom", false},
	{"http://example.org/elements/atom", false},
}

func TestIsURIRef(t *testing.T) {
	for _, tt := range IsURIRefTests {
		out := IsURIRef(tt.in)
		if out != tt.out {
			t.Errorf("IsURIRef(%s) => %t, want: %t", tt.in, out, tt.out)
		}
	}
}

var IsLiteralTests = []struct {
	in  string
	out bool
}{
	{"<http://example.org/show/218>", false},
	{"<http://example.org/elements/atomicNumber>", false},
	{"<http://example.org/elements/atom", false},
	{"http://example.org/elements/atom", false},
	{"\"atom\"", true},
}

func TestIsLiteral(t *testing.T) {
	for _, tt := range IsLiteralTests {
		out := IsLiteral(tt.in)
		if out != tt.out {
			t.Errorf("IsLiteral(%s) => %t, want: %t", tt.in, out, tt.out)
		}
	}
}

var IsLiteralLanguageTests = []struct {
	in   string
	lang string
	out  bool
}{
	{"<http://example.org/show/218>", "", false},
	{"<http://example.org/elements/atomicNumber>", "", false},
	{"<http://example.org/elements/atom", "", false},
	{"http://example.org/elements/atom", "", false},
	{"\"atom\"", "", true},
	{"\"atom\"", "en", true},
	{"\"atom\"@en", "en", true},
	{"\"atom\"@fr", "en", false},
}

func TestIsLiteralLanguage(t *testing.T) {
	for _, tt := range IsLiteralLanguageTests {
		out := IsLiteralLanguage(tt.in, tt.lang)
		if out != tt.out {
			t.Errorf("IsLiteral(%s, %s) => %t, want: %t", tt.in, tt.lang, out, tt.out)
		}
	}
}

var IsNamedNodeTests = []struct {
	in  string
	out bool
}{
	{"<http://example.org/show/218>", false},
	{"<http://example.org/elements/atomicNumber>", false},
	{"<http://example.org/elements/atom", false},
	{"http://example.org/elements/atom", false},
	{"\"atom\"", false},
	{"\"atom\"", false},
	{"\"atom\"@en", false},
	{"\"atom\"@fr", false},
	{"_:sa9df86sdf68", true},
}

func TestIsNamedNode(t *testing.T) {
	for _, tt := range IsNamedNodeTests {
		out := IsNamedNode(tt.in)
		if out != tt.out {
			t.Errorf("IsNamedNode(%s) => %t, want: %t", tt.in, out, tt.out)
		}
	}
}

var ParseRulesTests = []struct {
	in  string
	out []Rule
	err error
}{
	{`a hello
      b world`,
		[]Rule{Rule{Prefix: "hello", Shortcut: "a"},
			Rule{Prefix: "world", Shortcut: "b"}},
		nil},

	{`a hello
      // just a comment  
      b world`,
		[]Rule{Rule{Prefix: "hello", Shortcut: "a"},
			Rule{Prefix: "world", Shortcut: "b"}},
		nil},

	{`a hello
      # just a comment

      b world`,
		[]Rule{Rule{Prefix: "hello", Shortcut: "a"},
			Rule{Prefix: "world", Shortcut: "b"}},
		nil},

	{`a hello

      // do not mix, unless you have to
      # just a comment
      
      b world`,
		[]Rule{Rule{Prefix: "hello", Shortcut: "a"},
			Rule{Prefix: "world", Shortcut: "b"}},
		nil},

	{`a

      // do not mix, unless you have to
      # just a comment
      
      b world`,
		[]Rule{},
		errors.New("Broken rule: a")},
}

func TestParseRules(t *testing.T) {
	for _, tt := range ParseRulesTests {
		out, err := ParseRules(tt.in)
		if err != nil && err.Error() != tt.err.Error() {
			t.Errorf("ParseRules(%s) error mismatch => %s, want: %v", tt.in, err, tt.err)
		} else {

		}
		if err == nil && !reflect.DeepEqual(out, tt.out) {
			t.Errorf("ParseRules(%s) => %+v, want: %+v", tt.in, out, tt.out)
		}
	}
}
