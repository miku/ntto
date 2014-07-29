package ntto

import (
	"encoding/xml"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const AppVersion = "0.2"

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
			cmd := fmt.Sprintf("sed -e '%s' < '%s'", strings.Join(commands, "; "), in)
			replacements = append(replacements, cmd)
		} else {
			cmd := fmt.Sprintf("sed -e '%s'", strings.Join(commands, "; "))
			replacements = append(replacements, cmd)
		}

	}
	return strings.Join(replacements, " | ")
}
