package ntto

import (
	"errors"
	"reflect"
	"testing"
)

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
		errors.New("broken rule: a")},
}

func TestParseRules(t *testing.T) {
	for _, tt := range ParseRulesTests {
		out, err := ParseRules(tt.in)
		if err != nil && err.Error() != tt.err.Error() {
			t.Errorf("ParseRules(%s) error mismatch => %s, want: %v", tt.in, err, tt.err)
		} else {
			// pass
		}
		if err == nil && !reflect.DeepEqual(out, tt.out) {
			t.Errorf("ParseRules(%s) => %+v, want: %+v", tt.in, out, tt.out)
		}
	}
}

var PartitionRulesTests = []struct {
	in  []Rule
	p   int
	out [][]Rule
}{
	{
		[]Rule{Rule{Shortcut: "a", Prefix: "aaaa"},
			Rule{Shortcut: "b", Prefix: "bbbb"}},
		2,
		[][]Rule{
			[]Rule{Rule{Shortcut: "a", Prefix: "aaaa"}},
			[]Rule{Rule{Shortcut: "b", Prefix: "bbbb"}}},
	},
	{
		[]Rule{Rule{Shortcut: "a", Prefix: "aaaa"},
			Rule{Shortcut: "b", Prefix: "bbbb"}},
		1,
		[][]Rule{[]Rule{Rule{Shortcut: "a", Prefix: "aaaa"}, Rule{Shortcut: "b", Prefix: "bbbb"}}},
	},
	{
		[]Rule{Rule{Shortcut: "a", Prefix: "aaaa"},
			Rule{Shortcut: "b", Prefix: "bbbb"},
			Rule{Shortcut: "c", Prefix: "cccc"}},
		3,
		[][]Rule{
			[]Rule{Rule{Shortcut: "a", Prefix: "aaaa"}},
			[]Rule{Rule{Shortcut: "b", Prefix: "bbbb"}},
			[]Rule{Rule{Shortcut: "c", Prefix: "cccc"}},
		},
	},
	{
		[]Rule{Rule{Shortcut: "a", Prefix: "aaaa"},
			Rule{Shortcut: "b", Prefix: "bbbb"}},
		5,
		[][]Rule{
			[]Rule{Rule{Shortcut: "a", Prefix: "aaaa"}},
			[]Rule{Rule{Shortcut: "b", Prefix: "bbbb"}}},
	},
}

func TestPartitionRules(t *testing.T) {
	for _, tt := range PartitionRulesTests {
		out := PartitionRules(tt.in, tt.p)
		if !reflect.DeepEqual(out, tt.out) {
			t.Errorf("PartitionRules(%+v) => %+v, want: %+v", tt.in, out, tt.out)
		}
	}
}

var SedifyTests = []struct {
	rules []Rule
	p     int
	in    string
	out   string
}{
	{
		[]Rule{Rule{Shortcut: "a", Prefix: "aaaa"},
			Rule{Shortcut: "b", Prefix: "bbbb"}},
		2,
		"",
		"LANG=C perl -lnpe 's@aaaa@a:@g' | LANG=C perl -lnpe 's@bbbb@b:@g'",
	},
	{
		[]Rule{Rule{Shortcut: "a", Prefix: "aaaa"},
			Rule{Shortcut: "b", Prefix: "bbbb"}},
		1,
		"",
		"LANG=C perl -lnpe 's@aaaa@a:@g; s@bbbb@b:@g'",
	},
	{
		[]Rule{Rule{Shortcut: "a", Prefix: "aaaa"},
			Rule{Shortcut: "b", Prefix: "bbbb"}},
		1,
		"hello.txt",
		"LANG=C perl -lnpe 's@aaaa@a:@g; s@bbbb@b:@g' < 'hello.txt'",
	},
	{
		[]Rule{Rule{Shortcut: "a", Prefix: "aaaa"},
			Rule{Shortcut: "b", Prefix: "bbbb"}},
		2,
		"hello.txt",
		"LANG=C perl -lnpe 's@aaaa@a:@g' < 'hello.txt' | LANG=C perl -lnpe 's@bbbb@b:@g'",
	},
	{
		[]Rule{Rule{Shortcut: "a", Prefix: "aaaa"},
			Rule{Shortcut: "b", Prefix: "bbbb"},
			Rule{Shortcut: "c", Prefix: "cccc"},
			Rule{Shortcut: "d", Prefix: "dddd"},
			Rule{Shortcut: "e", Prefix: "eeee"},
			Rule{Shortcut: "f", Prefix: "ffff"}},
		2,
		"hello.txt",
		"LANG=C perl -lnpe 's@aaaa@a:@g; s@cccc@c:@g; s@eeee@e:@g' < 'hello.txt' | LANG=C perl -lnpe 's@bbbb@b:@g; s@dddd@d:@g; s@ffff@f:@g'",
	},
	{
		[]Rule{Rule{Shortcut: "a", Prefix: "aaaa"},
			Rule{Shortcut: "b", Prefix: "bbbb"},
			Rule{Shortcut: "c", Prefix: "cccc"},
			Rule{Shortcut: "d", Prefix: "dddd"},
			Rule{Shortcut: "e", Prefix: "eeee"},
			Rule{Shortcut: "f", Prefix: "ffff"}},
		4,
		"hello.txt",
		"LANG=C perl -lnpe 's@aaaa@a:@g; s@eeee@e:@g' < 'hello.txt' | LANG=C perl -lnpe 's@bbbb@b:@g; s@ffff@f:@g' | LANG=C perl -lnpe 's@cccc@c:@g' | LANG=C perl -lnpe 's@dddd@d:@g'",
	},
}

func TestSedify(t *testing.T) {
	for _, tt := range SedifyTests {
		out := Sedify(tt.rules, tt.p, tt.in)
		if out != tt.out {
			t.Errorf("Sedify(%+v, %d, %s) => %+v, want: %+v", tt.rules, tt.p, tt.in, out, tt.out)
		}
	}
}

var ParseNTripleTests = []struct {
	in  string
	out Triple
}{
	{`<http://d-nb.info/gnd/1-2> <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://d-nb.info/standards/elementset/gnd#SeriesOfConferenceOrEvent> .`,
		Triple{Subject: "http://d-nb.info/gnd/1-2",
			Predicate: "http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
			Object:    "http://d-nb.info/standards/elementset/gnd#SeriesOfConferenceOrEvent"}},
	{`a b c .`,
		Triple{Subject: "a", Predicate: "b", Object: "c"}},
	{`a b "the deep blue c" .`,
		Triple{Subject: "a", Predicate: "b", Object: "the deep blue c"}},
	{`a <b> "the deep blue c" .`,
		Triple{Subject: "a", Predicate: "b", Object: "the deep blue c"}},
	{`<a> <b> "the deep blue c" .`,
		Triple{Subject: "a", Predicate: "b", Object: "the deep blue c"}},
	{`<a> <b> <the deep blue c> .`,
		Triple{Subject: "a", Predicate: "b", Object: "the deep blue c"}},
	{`<a> <b> <the deep blue c>`,
		Triple{Subject: "a", Predicate: "b", Object: "the deep blue c"}},
	{`<a> <b> <the deep blue c>`,
		Triple{Subject: "a", Predicate: "b", Object: "the deep blue c"}},
	{`<a>    <b>  <the         deep blue c>`,
		Triple{Subject: "a", Predicate: "b", Object: "the deep blue c"}},
}

func TestParseNTriple(t *testing.T) {
	for _, tt := range ParseNTripleTests {
		out, _ := ParseNTriple(tt.in)
		if *out != tt.out {
			t.Errorf("ParseNTriple(%s) => %+v, want: %+v", tt.in, out, tt.out)
		}
	}
}
