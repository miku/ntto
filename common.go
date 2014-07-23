package ntto

import (
	"encoding/xml"
	"fmt"
	"strings"
)

const AppVersion = "0.2"

type Triple struct {
	XMLName   xml.Name `json:"-" xml:"t"`
	Subject   string   `json:"s" xml:"s"`
	Predicate string   `json:"s" xml:"s"`
	Object    string   `json:"s" xml:"s"`
}

type NamespaceAbbreviation struct {
	Namespace    string
	Abbreviation string
}

func (na NamespaceAbbreviation) String() string {
	return fmt.Sprintf("%s: %s", na.Abbreviation, na.Namespace)
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
