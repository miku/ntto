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
)

const AppVersion = "1.0.7"

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

func isLiteralLanguage(s, language string) bool {
	if !strings.Contains(s, "@") {
		return true
	} else {
		return strings.Contains(s, "@"+language)
	}
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

func Convert(fileName string,
	rules []Rule,
	format string,
	language string,
	ignore bool) (err error) {
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
		go Worker(queue, triples, rules, language, ignore)
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

	reader := bufio.NewReader(file)
	for {
		b, _, err := reader.ReadLine()
		if err != nil || b == nil {
			break
		}
		line := string(b)
		queue <- &line
	}

	// kill workers
	for n := 0; n < runtime.NumCPU(); n++ {
		queue <- nil
	}
	close(triples)
	return
}

// Worker converts NTriple to triples and sends them on the triples channel
func Worker(queue chan *string,
	triples chan *Triple,
	rules []Rule,
	language string,
	ignore bool) {
	var line *string
	for {
		line = <-queue
		if line == nil {
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
			if !ignore {
				os.Exit(1)
				break
			}
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
		if language != "" && !isLiteralLanguage(o, language) {
			continue
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

	# dbpedia languages (more below)
	dbp.de   http://de.dbpedia.org/resource/

	dbpopp http://dbpedia.org/ontology/PopulatedPlace/

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
	prov    http://www.w3.org/ns/prov#

	# more dbpedia languages w/ > 100k pages
	dbp.fr   http://fr.dbpedia.org/resource/
	dbp.en   http://en.dbpedia.org/resource/
	dbp.es   http://es.dbpedia.org/resource/
	dbp.it   http://it.dbpedia.org/resource/
	dbp.nl   http://nl.dbpedia.org/resource/
	dbp.ru   http://ru.dbpedia.org/resource/
	dbp.sv   http://sv.dbpedia.org/resource/
	dbp.pl   http://pl.dbpedia.org/resource/
	dbp.ja   http://ja.dbpedia.org/resource/
	dbp.pt   http://pt.dbpedia.org/resource/
	dbp.ar   http://ar.dbpedia.org/resource/
	dbp.zh   http://zh.dbpedia.org/resource/
	dbp.uk   http://uk.dbpedia.org/resource/
	dbp.ca   http://ca.dbpedia.org/resource/
	dbp.no   http://no.dbpedia.org/resource/
	dbp.fi   http://fi.dbpedia.org/resource/
	dbp.cs   http://cs.dbpedia.org/resource/
	dbp.hu   http://hu.dbpedia.org/resource/
	dbp.tr   http://tr.dbpedia.org/resource/
	dbp.ro   http://ro.dbpedia.org/resource/
	dbp.sw   http://sw.dbpedia.org/resource/
	dbp.ko   http://ko.dbpedia.org/resource/
	dbp.kk   http://kk.dbpedia.org/resource/
	dbp.vi   http://vi.dbpedia.org/resource/
	dbp.da   http://da.dbpedia.org/resource/
	dbp.eo   http://eo.dbpedia.org/resource/
	dbp.sr   http://sr.dbpedia.org/resource/
	dbp.id   http://id.dbpedia.org/resource/
	dbp.lt   http://lt.dbpedia.org/resource/
	dbp.vo   http://vo.dbpedia.org/resource/
	dbp.sk   http://sk.dbpedia.org/resource/
	dbp.he   http://he.dbpedia.org/resource/
	dbp.fa   http://fa.dbpedia.org/resource/
	dbp.bg   http://bg.dbpedia.org/resource/
	dbp.sl   http://sl.dbpedia.org/resource/
	dbp.eu   http://eu.dbpedia.org/resource/
	dbp.war   http://war.dbpedia.org/resource/
	dbp.et   http://et.dbpedia.org/resource/
	dbp.hr   http://hr.dbpedia.org/resource/
	dbp.ms   http://ms.dbpedia.org/resource/
	dbp.hi   http://hi.dbpedia.org/resource/
	dbp.sh   http://sh.dbpedia.org/resource/

	dbpwp.de   http://de.dbpedia.org/property/wikiPage
	dbpwp.fr   http://fr.dbpedia.org/property/wikiPage
	dbpwp.en   http://en.dbpedia.org/property/wikiPage
	dbpwp.es   http://es.dbpedia.org/property/wikiPage
	dbpwp.it   http://it.dbpedia.org/property/wikiPage
	dbpwp.nl   http://nl.dbpedia.org/property/wikiPage
	dbpwp.ru   http://ru.dbpedia.org/property/wikiPage
	dbpwp.sv   http://sv.dbpedia.org/property/wikiPage
	dbpwp.pl   http://pl.dbpedia.org/property/wikiPage
	dbpwp.ja   http://ja.dbpedia.org/property/wikiPage
	dbpwp.pt   http://pt.dbpedia.org/property/wikiPage
	dbpwp.ar   http://ar.dbpedia.org/property/wikiPage
	dbpwp.zh   http://zh.dbpedia.org/property/wikiPage
	dbpwp.uk   http://uk.dbpedia.org/property/wikiPage
	dbpwp.ca   http://ca.dbpedia.org/property/wikiPage
	dbpwp.no   http://no.dbpedia.org/property/wikiPage
	dbpwp.fi   http://fi.dbpedia.org/property/wikiPage
	dbpwp.cs   http://cs.dbpedia.org/property/wikiPage
	dbpwp.hu   http://hu.dbpedia.org/property/wikiPage
	dbpwp.tr   http://tr.dbpedia.org/property/wikiPage
	dbpwp.ro   http://ro.dbpedia.org/property/wikiPage
	dbpwp.sw   http://sw.dbpedia.org/property/wikiPage
	dbpwp.ko   http://ko.dbpedia.org/property/wikiPage
	dbpwp.kk   http://kk.dbpedia.org/property/wikiPage
	dbpwp.vi   http://vi.dbpedia.org/property/wikiPage
	dbpwp.da   http://da.dbpedia.org/property/wikiPage
	dbpwp.eo   http://eo.dbpedia.org/property/wikiPage
	dbpwp.sr   http://sr.dbpedia.org/property/wikiPage
	dbpwp.id   http://id.dbpedia.org/property/wikiPage
	dbpwp.lt   http://lt.dbpedia.org/property/wikiPage
	dbpwp.vo   http://vo.dbpedia.org/property/wikiPage
	dbpwp.sk   http://sk.dbpedia.org/property/wikiPage
	dbpwp.he   http://he.dbpedia.org/property/wikiPage
	dbpwp.fa   http://fa.dbpedia.org/property/wikiPage
	dbpwp.bg   http://bg.dbpedia.org/property/wikiPage
	dbpwp.sl   http://sl.dbpedia.org/property/wikiPage
	dbpwp.eu   http://eu.dbpedia.org/property/wikiPage
	dbpwp.war   http://war.dbpedia.org/property/wikiPage
	dbpwp.et   http://et.dbpedia.org/property/wikiPage
	dbpwp.hr   http://hr.dbpedia.org/property/wikiPage
	dbpwp.ms   http://ms.dbpedia.org/property/wikiPage
	dbpwp.hi   http://hi.dbpedia.org/property/wikiPage
	dbpwp.sh   http://sh.dbpedia.org/property/wikiPage

	dbpp.de   http://de.dbpedia.org/property/
	dbpp.fr   http://fr.dbpedia.org/property/
	dbpp.en   http://en.dbpedia.org/property/
	dbpp.es   http://es.dbpedia.org/property/
	dbpp.it   http://it.dbpedia.org/property/
	dbpp.nl   http://nl.dbpedia.org/property/
	dbpp.ru   http://ru.dbpedia.org/property/
	dbpp.sv   http://sv.dbpedia.org/property/
	dbpp.pl   http://pl.dbpedia.org/property/
	dbpp.ja   http://ja.dbpedia.org/property/
	dbpp.pt   http://pt.dbpedia.org/property/
	dbpp.ar   http://ar.dbpedia.org/property/
	dbpp.zh   http://zh.dbpedia.org/property/
	dbpp.uk   http://uk.dbpedia.org/property/
	dbpp.ca   http://ca.dbpedia.org/property/
	dbpp.no   http://no.dbpedia.org/property/
	dbpp.fi   http://fi.dbpedia.org/property/
	dbpp.cs   http://cs.dbpedia.org/property/
	dbpp.hu   http://hu.dbpedia.org/property/
	dbpp.tr   http://tr.dbpedia.org/property/
	dbpp.ro   http://ro.dbpedia.org/property/
	dbpp.sw   http://sw.dbpedia.org/property/
	dbpp.ko   http://ko.dbpedia.org/property/
	dbpp.kk   http://kk.dbpedia.org/property/
	dbpp.vi   http://vi.dbpedia.org/property/
	dbpp.da   http://da.dbpedia.org/property/
	dbpp.eo   http://eo.dbpedia.org/property/
	dbpp.sr   http://sr.dbpedia.org/property/
	dbpp.id   http://id.dbpedia.org/property/
	dbpp.lt   http://lt.dbpedia.org/property/
	dbpp.vo   http://vo.dbpedia.org/property/
	dbpp.sk   http://sk.dbpedia.org/property/
	dbpp.he   http://he.dbpedia.org/property/
	dbpp.fa   http://fa.dbpedia.org/property/
	dbpp.bg   http://bg.dbpedia.org/property/
	dbpp.sl   http://sl.dbpedia.org/property/
	dbpp.eu   http://eu.dbpedia.org/property/
	dbpp.war   http://war.dbpedia.org/property/
	dbpp.et   http://et.dbpedia.org/property/
	dbpp.hr   http://hr.dbpedia.org/property/
	dbpp.ms   http://ms.dbpedia.org/property/
	dbpp.hi   http://hi.dbpedia.org/property/
	dbpp.sh   http://sh.dbpedia.org/property/

	atom   http://www.w3.org/2005/Atom
	address http://schemas.talis.com/2005/address/schema#
	admin   http://webns.net/mvcb/
	atom    http://atomowl.org/ontologies/atomrdf#
	aws http://soap.amazon.com/
	b3s http://b3s.openlinksw.com/
	batch   http://schemas.google.com/gdata/batch/
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
	ff  http://api.friendfeed.com/2008/03/
	fn  http://www.w3.org/2005/xpath-functions/#
	g   http://base.google.com/ns/1.0/
	gb  http://www.openlinksw.com/schemas/google-base#
	gd  http://schemas.google.com/g/2005/
	geo http://www.w3.org/2003/01/geo/wgs84_pos#
	geonames    http://www.geonames.org/ontology#
	georss  http://www.georss.org/georss/
	gml http://www.opengis.net/gml/
	go  http://purl.org/obo/owl/GO#
	grs http://www.georss.org/georss/
	hlisting    http://www.openlinksw.com/schemas/hlisting/
	hoovers http://wwww.hoovers.com/
	hrev    http:/www.purl.org/stuff/hrev#
	ical    http://www.w3.org/2002/12/cal/ical#
	ir  http://web-semantics.org/ns/image-regions/
	itunes  http://www.itunes.com/DTDs/Podcast-1.0.dtd
	lgv http://linkedgeodata.org/vocabulary#
	link    http://www.xbrl.org/2003/linkbase/
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
	r   http://backend.userland.com/rss2/
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
	virt    http://www.openlinksw.com/virtuoso/xslt/
	virtcxml    http://www.openlinksw.com/schemas/virtcxml#
	virtrdf http://www.openlinksw.com/schemas/virtrdf#
	void    http://rdfs.org/ns/void#
	wb  http://www.worldbank.org/
	wf  http://www.w3.org/2005/01/wf/flow#
	wfw http://wellformedweb.org/CommentAPI/
	xf  http://www.w3.org/2004/07/xpath-functions/
	xfn http://gmpg.org/xfn/11#
	xhtml   http://www.w3.org/1999/xhtml/
	xhv http://www.w3.org/1999/xhtml/vocab#
	xi  http://www.xbrl.org/2003/instance/
	xml http://www.w3.org/XML/1998/namespace/
	xn  http://www.ning.com/atom/1.0/
	xsd http://www.w3.org/2001/XMLSchema#
	xsl10   http://www.w3.org/XSL/Transform/1.0
	xsl1999 http://www.w3.org/1999/XSL/Transform/
	xslwd   http://www.w3.org/TR/WD-xsl/
	y   urn:yahoo:maps
	yago    http://dbpedia.org/class/yago/
	yt  http://gdata.youtube.com/schemas/2007/
	zem http://s.zemanta.com/ns#
	`

	runtime.GOMAXPROCS(runtime.NumCPU())

	format := flag.String("f", "json", "output format (json, xml, tsv)")
	abbreviate := flag.Bool("a", false, "abbreviate triples")
	profile := flag.Bool("p", false, "cpu profile")
	dumpRules := flag.Bool("d", false, "dump rules as TSV to stdout")
	version := flag.Bool("v", false, "prints current version and exits")
	ignore := flag.Bool("i", false, "ignore any conversion error")
	language := flag.String("l", "", "only keep literals of the given language")

	flag.Parse()

	// slice of Rule holds the rewrite table
	var rules []Rule
	var err error

	if *version {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	if *dumpRules {
		rules, err = parseRules(table)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		for _, rule := range rules {
			fmt.Printf("%s\t%s\n", rule.Shortcut, rule.Prefix)
		}
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

	err = Convert(fileName, rules, *format, *language, *ignore)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	// profiling
	if *profile {
		pprof.StopCPUProfile()
	}
}
