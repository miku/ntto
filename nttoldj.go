package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strings"
)

type Triple struct {
	Subject   string `json:"s"`
	Predicate string `json:"p"`
	Object    string `json:"o"`
}

type Rule struct {
	Pattern  *regexp.Regexp
	Shortcut string
}

func isURIRef(s string) bool {
	return strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">")
}

func isLiteral(s string) bool {
	return strings.HasPrefix(s, "\"")
}

func isNamedNode(s string) bool {
	return strings.HasPrefix(s, "_:")
}

// stripChars makes s, p, o strings unembellished
func stripChars(s string) string {
	if isURIRef(s) {
		s = strings.Trim(s, "<>")
	} else if isLiteral(s) {
		if strings.Contains(s, "@") {
			parts := strings.Split(s, "@")
			s = strings.Join(parts[:len(parts)-1], "")
		}
		if strings.Contains(s, "^^") {
			parts := strings.Split(s, "^^")
			s = strings.Join(parts[:len(parts)-1], "")
		}
		s = strings.Trim(s, "\"")
	}
	return s
}

// readLines reads a whole file into the memory
func readLines(path string) (lines []string, err error) {
	var (
		file   *os.File
		part   []byte
		prefix bool
	)
	if file, err = os.Open(path); err != nil {
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	buffer := bytes.NewBuffer(make([]byte, 0))
	for {
		if part, prefix, err = reader.ReadLine(); err != nil {
			break
		}
		buffer.Write(part)
		if !prefix {
			lines = append(lines, buffer.String())
			buffer.Reset()
		}
	}
	if err == io.EOF {
		err = nil
	}
	return
}

// parseString takes a string, parses out rules and adds them
func parseRules(s string) []Rule {
	var rules []Rule
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			fmt.Fprintf(os.Stderr, "broken rules: %s", line)
			os.Exit(1)
		}
		pattern := regexp.MustCompile(fields[0])
		rule := Rule{Pattern: pattern, Shortcut: fields[1]}
		rules = append(rules, rule)
	}
	return rules
}

// applyRules takes a string and applies the rules
func applyRules(s string, rules []Rule) string {
	// could optimize this routine on the fly by JIT reordering the rules
	for _, rule := range rules {
		matched := rule.Pattern.FindStringSubmatch(s)
		if len(matched) > 0 {
			s = strings.Replace(rule.Shortcut, "$1", matched[1], -1)
			break
		}
	}
	return s
}

func Convert(fileName string, rules []Rule) {
	// lines will be sent down queue chan
	queue := make(chan *string)
	triples := make(chan *Triple)

	// start writer
	go TripleWriter(triples)

	// start workers
	for i := 0; i < runtime.NumCPU(); i++ {
		go Worker(i, queue, triples, rules)
	}

	var file *os.File
	if fileName == "-" {
		file = os.Stdin
	} else {
		var err error
		file, err = os.Open(fileName)
		defer file.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "no such file or directory\n")
			os.Exit(1)
		}
	}

	// SCANNER
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var line = scanner.Text()
		queue <- &line
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "reading input:", err)
	}

	// kill workers
	for n := 0; n < runtime.NumCPU(); n++ {
		queue <- nil
	}
	// kill writer
	triples <- nil
}

// TripleWriter dumps a stream of triples to
func TripleWriter(triples chan *Triple) {
	var triple *Triple
	for {
		triple = <-triples
		if triples == nil {
			break
		}
		b, err := json.Marshal(triple)
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshalling error:", err)
			os.Exit(1)
		}
		fmt.Println(string(b))
	}
}

// Worker converts NT to JSON
func Worker(id int, queue chan *string, triples chan *Triple, rules []Rule) {
	var line *string
	for {
		line = <-queue
		if line == nil {
			break
		} else {

			trimmed := strings.TrimSpace(*line)

			// ignore comments
			if strings.HasPrefix(trimmed, "#") {
				continue
			}
			words := strings.Fields(trimmed)

			var s, p, o string

			if len(words) < 3 {
				fmt.Fprintf(os.Stderr, "broken input:", trimmed)
				os.Exit(1)
			}

			if len(words) == 4 || len(words) == 3 {
				s = words[0]
				p = words[1]
				o = words[2]
			}
			// take care of possible spaces in Object
			if len(words) > 4 {
				s = words[0]
				p = words[1]
				o = strings.Join(words[2:len(words)-1], " ")
			}
			// make things clean
			s = stripChars(s)
			p = stripChars(p)
			o = stripChars(o)

			// make things short
			s = applyRules(s, rules)
			p = applyRules(p, rules)
			o = applyRules(o, rules)

			// convert to json
			triple := Triple{Subject: s, Predicate: p, Object: o}
			triples <- &triple
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()

	table := `
	http://dbpedia.org/resource/(.+)                    dbpr:$1
	http://dbpedia.org/ontology/PopulatedPlace/(.+)     dbpl:$1
	http://dbpedia.org/ontology/(.+)                    dbpo:$1
	http://www.w3.org/1999/02/22-rdf-syntax-ns#(.+)     rdf:$1
    http://www.w3.org/2000/01/rdf-schema#(.+)           rdfs:$1
	http://xmlns.com/foaf/0.1/(.+)                      foar:$1
	http://purl.org/dc/elements/1.1/(.+)                dc:$1
	`

	rules := parseRules(table)

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s FILE\n", os.Args[0])
		os.Exit(1)
	}
	fileName := flag.Args()[0]
	Convert(fileName, rules)
}
