package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"
)

const AppVersion = "1.0.2"

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

func (r Rule) String() string {
	return fmt.Sprintf("%+v", r)
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

// stripChars makes s, p, o strings "unembellished"
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

// parseRules takes a string, parses out rules and returns them as slice
func parseRules(s string) (rules []Rule, err error) {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			err = errors.New(fmt.Sprintf("broken rule: %s", line))
			break
		}
		rules = append(rules, Rule{Prefix: fields[1], Shortcut: fields[0]})
	}
	return
}

// applyRules takes a string and applies the rules
func applyRules(s string, rules []Rule) string {
	for _, rule := range rules {
		if strings.HasPrefix(s, rule.Prefix) {
			s = strings.Replace(s, rule.Prefix, rule.Shortcut+":", -1)
		}
	}
	return s
}

func Convert(fileName string, rules []Rule, format string) (err error) {
	// lines will be sent down queue channel
	queue := make(chan *string)
	// send triples down this channel
	triples := make(chan *Triple)

	// start writer
	if format == "json" {
		go JsonTripleWriter(triples)
	} else if format == "xml" {
		go XmlTripleWriter(triples)
	} else if format == "tsv" {
		go TSVTripleWriter(triples)
	} else {
		err = errors.New(fmt.Sprintf("unknown format: %s\n", format))
		return
	}

	// start workers
	for i := 0; i < runtime.NumCPU(); i++ {
		go Worker(queue, triples, rules)
	}

	var file *os.File
	if fileName == "-" {
		file = os.Stdin
	} else {
		file, err = os.Open(fileName)
		defer file.Close()
		if err != nil {
			return
		}
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var line = scanner.Text()
		queue <- &line
	}

	err = scanner.Err()

	// kill workers
	for n := 0; n < runtime.NumCPU(); n++ {
		queue <- nil
	}
	close(triples)
	return
}

// Worker converts NTriple to triples and sends them on the triples channel
func Worker(queue chan *string, triples chan *Triple, rules []Rule) {
	var line *string
	for {
		line = <-queue
		if line == nil {
			time.Sleep(1000 * time.Millisecond)
			break
		}
		trimmed := strings.TrimSpace(*line)

		// ignore comments
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		words := strings.Fields(trimmed)
		var s, p, o string

		if len(words) < 3 {
			fmt.Fprintf(os.Stderr, "broken input: %s\n", words)
			os.Exit(1)
			break
		} else if len(words) == 4 || len(words) == 3 {
			s = words[0]
			p = words[1]
			o = words[2]
		} else if len(words) > 4 {
			// take care of possible spaces in Object
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

		triples <- &Triple{Subject: s, Predicate: p, Object: o}
	}
}

// JsonTripleWriter dumps a stream of triples to json
func JsonTripleWriter(triples chan *Triple) {
	for triple := range triples {
		b, err := json.Marshal(triple)
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshalling error:", err)
			os.Exit(1)
		}
		fmt.Println(string(b))
	}
}

// XmlTripleWriter dumps a stream of triples to xml
func XmlTripleWriter(triples chan *Triple) {
	for triple := range triples {
		b, err := xml.Marshal(triple)
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshalling error:", err)
			os.Exit(1)
		}
		fmt.Println(string(b))
	}
}

func TSVTripleWriter(triples chan *Triple) {
	for triple := range triples {
		fmt.Printf("%s\t%s\t%s\n", triple.Subject, triple.Predicate, triple.Object)
	}
}

func main() {
	// TODO: allow this to be read from file
	table := `
	dbp http://dbpedia.org/resource/
	dbpopp http://dbpedia.org/ontology/PopulatedPlace/

	# abbreviates dbpo:wikiPageWikiLink to dbpow:Link
	dbpow http://dbpedia.org/ontology/wikiPageWiki

	# abbreviate things like wikiPageRedirects to dbpowp:Redirects
	dbpowp   http://dbpedia.org/ontology/wikiPage
	
	dbpo http://dbpedia.org/ontology/
	dbpp http://dbpedia.org/property/

	foaf    http://xmlns.com/foaf/0.1/
	rdf http://www.w3.org/1999/02/22-rdf-syntax-ns#
	rdfs    http://www.w3.org/2000/01/rdf-schema#
	schema      http://schema.org/
	dc  http://purl.org/dc/elements/1.1/
	dcterms http://purl.org/dc/terms/

	rdfa    http://www.w3.org/ns/rdfa#
	rdfdf   http://www.openlinksw.com/virtrdf-data-formats#
	umbel   http://umbel.org/umbel#
	umbel-ac    http://umbel.org/umbel/ac/
	umbel-sc    http://umbel.org/umbel/sc/

	a   http://www.w3.org/2005/Atom
	address http://schemas.talis.com/2005/address/schema#
	admin   http://webns.net/mvcb/
	atom    http://atomowl.org/ontologies/atomrdf#
	aws http://soap.amazon.com/
	b3s http://b3s.openlinksw.com/
	batch   http://schemas.google.com/gdata/batch
	bibo    http://purl.org/ontology/bibo/
	bugzilla    http://www.openlinksw.com/schemas/bugzilla#
	c   http://www.w3.org/2002/12/cal/icaltzd#
	category    http://dbpedia.org/resource/Category:
	cb  http://www.crunchbase.com/
	cc  http://web.resource.org/cc/
	content http://purl.org/rss/1.0/modules/content/
	cv  http://purl.org/captsolo/resume-rdf/0.2/cv#
	cvbase  http://purl.org/captsolo/resume-rdf/0.2/base#
	dawgt   http://www.w3.org/2001/sw/DataAccess/tests/test-dawg#
	digg    http://digg.com/docs/diggrss/
	ebay    urn:ebay:apis:eBLBaseComponents
	enc http://purl.oclc.org/net/rss_2.0/enc#
	exif    http://www.w3.org/2003/12/exif/ns/
	fb  http://api.facebook.com/1.0/
	fbase   http://rdf.freebase.com/ns/
	ff  http://api.friendfeed.com/2008/03
	fn  http://www.w3.org/2005/xpath-functions/#
	g   http://base.google.com/ns/1.0
	gb  http://www.openlinksw.com/schemas/google-base#
	gd  http://schemas.google.com/g/2005
	geo http://www.w3.org/2003/01/geo/wgs84_pos#
	geonames    http://www.geonames.org/ontology#
	georss  http://www.georss.org/georss
	gml http://www.opengis.net/gml
	go  http://purl.org/obo/owl/GO#
	grs http://www.georss.org/georss/
	hlisting    http://www.openlinksw.com/schemas/hlisting/
	hoovers http://wwww.hoovers.com/
	hrev    http:/www.purl.org/stuff/hrev#
	ical    http://www.w3.org/2002/12/cal/ical#
	ir  http://web-semantics.org/ns/image-regions
	itunes  http://www.itunes.com/DTDs/Podcast-1.0.dtd
	lgv http://linkedgeodata.org/vocabulary#
	link    http://www.xbrl.org/2003/linkbase
	lod http://lod.openlinksw.com/
	math    http://www.w3.org/2000/10/swap/math#
	media   http://search.yahoo.com/mrss/
	mesh    http://purl.org/commons/record/mesh/
	meta    urn:oasis:names:tc:opendocument:xmlns:meta:1.0
	mf  http://www.w3.org/2001/sw/DataAccess/tests/test-manifest#
	mmd http://musicbrainz.org/ns/mmd-1.0#
	mo  http://purl.org/ontology/mo/
	mql http://www.freebase.com/
	nci http://ncicb.nci.nih.gov/xml/owl/EVS/Thesaurus.owl#
	nfo http://www.semanticdesktop.org/ontologies/nfo/#
	ng  http://www.openlinksw.com/schemas/ning#
	nyt http://www.nytimes.com/
	oai http://www.openarchives.org/OAI/2.0/
	oai_dc  http://www.openarchives.org/OAI/2.0/oai_dc/
	obo http://www.geneontology.org/formats/oboInOwl#
	office  urn:oasis:names:tc:opendocument:xmlns:office:1.0
	oo  urn:oasis:names:tc:opendocument:xmlns:meta:1.0:
	openSearch  http://a9.com/-/spec/opensearchrss/1.0/
	opl http://www.openlinksw.com/schema/attribution#
	opl-gs  http://www.openlinksw.com/schemas/getsatisfaction/
	opl-meetup  http://www.openlinksw.com/schemas/meetup/
	opl-xbrl    http://www.openlinksw.com/schemas/xbrl/
	oplweb  http://www.openlinksw.com/schemas/oplweb#
	ore http://www.openarchives.org/ore/terms/
	owl http://www.w3.org/2002/07/owl#
	product http://www.buy.com/rss/module/productV2/
	protseq http://purl.org/science/protein/bysequence/
	r   http://backend.userland.com/rss2
	radio   http://www.radiopop.co.uk/
	rev http://purl.org/stuff/rev#
	review  http:/www.purl.org/stuff/rev#
	rss http://purl.org/rss/1.0/
	sc  http://purl.org/science/owl/sciencecommons/
	scovo   http://purl.org/NET/scovo#
	sf  urn:sobject.enterprise.soap.sforce.com
	sioc    http://rdfs.org/sioc/ns#
	sioct   http://rdfs.org/sioc/types#
	skos    http://www.w3.org/2004/02/skos/core#
	slash   http://purl.org/rss/1.0/modules/slash/
	stock   http://xbrlontology.com/ontology/finance/stock_market#
	twfy    http://www.openlinksw.com/schemas/twfy#
	uniprot http://purl.uniprot.org/
	usc http://www.rdfabout.com/rdf/schema/uscensus/details/100pct/
	v   http://www.openlinksw.com/xsltext/
	vcard   http://www.w3.org/2001/vcard-rdf/3.0#
	vcard2006   http://www.w3.org/2006/vcard/ns#
	vi  http://www.openlinksw.com/virtuoso/xslt/
	virt    http://www.openlinksw.com/virtuoso/xslt
	virtcxml    http://www.openlinksw.com/schemas/virtcxml#
	virtrdf http://www.openlinksw.com/schemas/virtrdf#
	void    http://rdfs.org/ns/void#
	wb  http://www.worldbank.org/
	wf  http://www.w3.org/2005/01/wf/flow#
	wfw http://wellformedweb.org/CommentAPI/
	xf  http://www.w3.org/2004/07/xpath-functions
	xfn http://gmpg.org/xfn/11#
	xhtml   http://www.w3.org/1999/xhtml
	xhv http://www.w3.org/1999/xhtml/vocab#
	xi  http://www.xbrl.org/2003/instance
	xml http://www.w3.org/XML/1998/namespace
	xn  http://www.ning.com/atom/1.0
	xsd http://www.w3.org/2001/XMLSchema#
	xsl10   http://www.w3.org/XSL/Transform/1.0
	xsl1999 http://www.w3.org/1999/XSL/Transform
	xslwd   http://www.w3.org/TR/WD-xsl
	y   urn:yahoo:maps
	yago    http://dbpedia.org/class/yago/
	yt  http://gdata.youtube.com/schemas/2007
	zem http://s.zemanta.com/ns#
	`

	runtime.GOMAXPROCS(runtime.NumCPU())

	format := flag.String("f", "json", "output format (json, xml, tsv)")
	abbreviate := flag.Bool("a", false, "abbreviate triples")
	profile := flag.Bool("p", false, "cpu profile")
	version := flag.Bool("v", false, "prints current version and exits")

	flag.Parse()

	// slice of Rule holds the rewrite table
	var rules []Rule
	var err error

	if *version {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	if *abbreviate {
		rules, err = parseRules(table)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
	}

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s FILE\n", os.Args[0])
		os.Exit(1)
	}

	fileName := flag.Args()[0]
	fmt.Fprintf(os.Stderr, "%d workers/%d rules\n", runtime.NumCPU(), len(rules))

	// profiling
	if *profile {
		file, err := os.Create("nttoldj.pprof")
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not create profile output\n")
			os.Exit(1)
		}
		_ = pprof.StartCPUProfile(file)
	}

	err = Convert(fileName, rules, *format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	// profiling
	if *profile {
		pprof.StopCPUProfile()
	}
}
