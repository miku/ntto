package ntto

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

const AppVersion = "0.3.4"

type Triple struct {
	XMLName   xml.Name `json:"-" xml:"t"`
	Subject   string   `json:"s" xml:"s"`
	Predicate string   `json:"p" xml:"p"`
	Object    string   `json:"o" xml:"o"`
}

type Rule struct {
	Prefix   string
	Shortcut string
}

func DumpRules(rules []Rule) string {
	var formatted []string
	for _, rule := range rules {
		formatted = append(formatted, rule.String())
	}
	sort.Strings(formatted)
	return strings.Join(formatted, "\n")
}

func (r Rule) String() string {
	return fmt.Sprintf("%s\t%s", r.Shortcut, r.Prefix)
}

func IsURIRef(s string) bool {
	return strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">")
}

func IsLiteral(s string) bool {
	return strings.HasPrefix(s, "\"")
}

func IsLiteralLanguage(s, language string) bool {
	if !IsLiteral(s) {
		return false
	}
	if !strings.Contains(s, "@") {
		return true
	} else {
		return strings.Contains(s, "@"+language)
	}
}

func IsNamedNode(s string) bool {
	return strings.HasPrefix(s, "_:")
}

// Simplistic NTriples parser
func ParseNTriple(line string) (*Triple, error) {
	line = strings.TrimSpace(line)
	words := strings.Fields(line)
	if len(words) < 3 {
		return nil, errors.New(fmt.Sprintf("Broken input: %s\n", words))
	}
	var s, p, o string

	s = words[0]
	p = words[1]

	if len(words) <= 4 {
		o = words[2]
	} else if len(words) > 4 {
		if strings.HasSuffix(line, ".") {
			o = strings.Join(words[2:len(words)-1], " ")
		} else {
			o = strings.Join(words[2:len(words)], " ")
		}
	}
	s = strings.Trim(s, "<>\"")
	p = strings.Trim(p, "<>\"")
	o = strings.Trim(o, "<>\"")
	triple := Triple{Subject: s, Predicate: p, Object: o}
	return &triple, nil
}

// ParseAbbreviations takes a string, parse the abbreviations and returns them as slice
func ParseRules(s string) ([]Rule, error) {
	var rules []Rule
	var err error
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			err = errors.New(fmt.Sprintf("Broken rule: %s", line))
			break
		}
		rules = append(rules, Rule{Prefix: fields[1], Shortcut: fields[0]})
	}
	return rules, err
}

// PartitionRules divides the rules slice into `count` partitions
func PartitionRules(rules []Rule, count int) [][]Rule {
	count = int(math.Min(float64(len(rules)), float64(count)))
	partitions := make([][]Rule, count)
	for i, rule := range rules {
		p := i % count
		partitions[p] = append(partitions[p], rule)
	}
	return partitions
}

// Turn rules into a sed command `in` as input, `out` as output filename
func Sedify(rules []Rule, p int, in string) string {
	return SedifyNull(rules, p, in, "<NULL>")
}

// Turn rules into a sed command `in` as input, `out` as output filename
func SedifyNull(rules []Rule, p int, in, null string) string {
	partitions := PartitionRules(rules, p)
	// 's@http://d-nb.info/gnd/@gnd:@g; s@http://d-nb.info/standards/elementset/gnd#@dnb:@g'
	var replacements []string
	for i, p := range partitions {
		commands := make([]string, len(p))
		for j, rule := range p {
			if rule.Shortcut == null {
				commands[j] = fmt.Sprintf("s@%s@@g", rule.Prefix)
			} else {
				commands[j] = fmt.Sprintf("s@%s@%s:@g", rule.Prefix, rule.Shortcut)
			}
		}
		if i == 0 && in != "" {
			cmd := fmt.Sprintf("LANG=C perl -lnpe '%s' < '%s'", strings.Join(commands, "; "), in)
			replacements = append(replacements, cmd)
		} else {
			cmd := fmt.Sprintf("LANG=C perl -lnpe '%s'", strings.Join(commands, "; "))
			replacements = append(replacements, cmd)
		}

	}
	return strings.Join(replacements, " | ")
}

func Replacify(rules []Rule, in string) string {
	return ReplacifyNull(rules, in, "<NULL>")
}

func ReplacifyNull(rules []Rule, in, null string) string {
	var buffer bytes.Buffer
	for _, rule := range rules {
		if rule.Shortcut == null {
			buffer.WriteString(fmt.Sprintf(" '%s' '' ", rule.Prefix))
		} else {
			buffer.WriteString(fmt.Sprintf(" '%s' '%s:' ", rule.Prefix, rule.Shortcut))
		}
	}
	return fmt.Sprintf("replace %s < %s", buffer.String(), in)
}
