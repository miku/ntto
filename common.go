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
