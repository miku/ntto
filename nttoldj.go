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

const AppVersion = "1.0.11"

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
	d http://dbpedia.org/resource/

	# dbpedia languages (more below)
	d.de   http://de.dbpedia.org/resource/

	do.pp http://dbpedia.org/ontology/PopulatedPlace/
	do.wp   http://dbpedia.org/ontology/wikiPage
	
	do http://dbpedia.org/ontology/
	dp.wp   http://dbpedia.org/property/wikiPage
	dp http://dbpedia.org/property/

	gnd          http://d-nb.info/gnd/
	dnb.es       http://d-nb.info/standards/elementset/gnd#
	dnb.area     http://d-nb.info/standards/vocab/gnd/geographic-area-code#
	dnb.voc      http://d-nb.info/standards/vocab/gnd/

	viaf    http://viaf.org/viaf/

	foaf    http://xmlns.com/foaf/0.1/
	rdf http://www.w3.org/1999/02/22-rdf-syntax-ns#
	rdfs    http://www.w3.org/2000/01/rdf-schema#
	schema      http://schema.org/
	dc  http://purl.org/dc/elements/1.1/
	dcterms http://purl.org/dc/terms/

	# freebase
	f.aa   http://rdf.freebase.com/ns/award.award_honor.
	f.ba   http://rdf.freebase.com/ns/base.articleindices.
	f.bb   http://rdf.freebase.com/ns/business.board_member.
	f.bc   http://rdf.freebase.com/ns/business.consumer_product.
	f.be   http://rdf.freebase.com/ns/business.employment_tenure.
	f.br   http://rdf.freebase.com/ns/base.rosetta.
	f.bss.fc   http://rdf.freebase.com/ns/base.schemastaging.food_concept.
	f.bss.ni   http://rdf.freebase.com/ns/base.schemastaging.nutrition_information.
	f.cd   http://rdf.freebase.com/ns/common.document.
	f.cid   http://rdf.freebase.com/ns/common.identity.
	f.cimg   http://rdf.freebase.com/ns/common.image.
	f.cl   http://rdf.freebase.com/ns/common.licensed_object.
	f.cn   http://rdf.freebase.com/ns/common.notable_for.
	f.ct   http://rdf.freebase.com/ns/common.topic.
	f.cw   http://rdf.freebase.com/ns/common.webpage.
	f.dg   http://rdf.freebase.com/ns/dataworld.gardening_hint.
	f.dm   http://rdf.freebase.com/ns/dataworld.mass_data_operation.
	f.dp   http://rdf.freebase.com/ns/freebase.domain_profile.
	f.ee   http://rdf.freebase.com/ns/education.education.
	f.ff   http://rdf.freebase.com/ns/fictional_universe.fictional_character.
	f.fv   http://rdf.freebase.com/ns/freebase.valuenotation.
	f.fo   http://rdf.freebase.com/ns/freebase.object_hints.
	f.mediacat   http://rdf.freebase.com/ns/media_common.cataloged_instance.
	f.gp   http://rdf.freebase.com/ns/government.political_district.
	f.ll   http://rdf.freebase.com/ns/location.location.
	f.lm   http://rdf.freebase.com/ns/location.mailing_address.
	f.med   http://rdf.freebase.com/ns/medicine.
	f.medd   http://rdf.freebase.com/ns/medicine.disease.
	f.medmdf   http://rdf.freebase.com/ns/medicine.manufactured_drug_form.
	f.meds   http://rdf.freebase.com/ns/medicine.symptom.
	f.mu   http://rdf.freebase.com/ns/measurement_unit.
	f.mudf  http://rdf.freebase.com/ns/measurement_unit.dated_float.
	f.mudi  http://rdf.freebase.com/ns/measurement_unit.dated_integer.
	f.mudmv http://rdf.freebase.com/ns/measurement_unit.dated_money_value.
	f.mumr  http://rdf.freebase.com/ns/measurement_unit.monetary_range.
	f.murs   http://rdf.freebase.com/ns/measurement_unit.rect_size.
	f.oo   http://rdf.freebase.com/ns/organization.organization.
	f.pp   http://rdf.freebase.com/ns/people.person.
	f.psls http://rdf.freebase.com/ns/protected_sites.listed_site.
	f.psnocsl http://rdf.freebase.com/ns/protected_sites.natural_or_cultural_site_listing.
	f.rc   http://rdf.freebase.com/ns/royalty.chivalric_order_membership.
	f.sp   http://rdf.freebase.com/ns/sports.pro_athlete.
	f.tc   http://rdf.freebase.com/ns/type.content.
	f.to   http://rdf.freebase.com/ns/type.object.
	f.tt   http://rdf.freebase.com/ns/type.type.

	# freebase 2
	f.m   http://rdf.freebase.com/ns/music.

	# generic freebase
	f      http://rdf.freebase.com/ns/
	fkey   http://rdf.freebase.com/key/

	rdfa    http://www.w3.org/ns/rdfa#
	rdfdf   http://www.openlinksw.com/virtrdf-data-formats#
	umbel   http://umbel.org/umbel#
	umbel-ac    http://umbel.org/umbel/ac/
	umbel-sc    http://umbel.org/umbel/sc/
	prov    http://www.w3.org/ns/prov#


	# more dbpedia languages w/ > 100k pages
	d.fr   http://fr.dbpedia.org/resource/
	d.en   http://en.dbpedia.org/resource/
	d.es   http://es.dbpedia.org/resource/
	d.it   http://it.dbpedia.org/resource/
	d.nl   http://nl.dbpedia.org/resource/
	d.ru   http://ru.dbpedia.org/resource/
	d.sv   http://sv.dbpedia.org/resource/
	d.pl   http://pl.dbpedia.org/resource/
	d.ja   http://ja.dbpedia.org/resource/
	d.pt   http://pt.dbpedia.org/resource/
	d.ar   http://ar.dbpedia.org/resource/
	d.zh   http://zh.dbpedia.org/resource/
	d.uk   http://uk.dbpedia.org/resource/
	d.ca   http://ca.dbpedia.org/resource/
	d.no   http://no.dbpedia.org/resource/
	d.fi   http://fi.dbpedia.org/resource/
	d.cs   http://cs.dbpedia.org/resource/
	d.hu   http://hu.dbpedia.org/resource/
	d.tr   http://tr.dbpedia.org/resource/
	d.ro   http://ro.dbpedia.org/resource/
	d.sw   http://sw.dbpedia.org/resource/
	d.ko   http://ko.dbpedia.org/resource/
	d.kk   http://kk.dbpedia.org/resource/
	d.vi   http://vi.dbpedia.org/resource/
	d.da   http://da.dbpedia.org/resource/
	d.eo   http://eo.dbpedia.org/resource/
	d.sr   http://sr.dbpedia.org/resource/
	d.id   http://id.dbpedia.org/resource/
	d.lt   http://lt.dbpedia.org/resource/
	d.vo   http://vo.dbpedia.org/resource/
	d.sk   http://sk.dbpedia.org/resource/
	d.he   http://he.dbpedia.org/resource/
	d.fa   http://fa.dbpedia.org/resource/
	d.bg   http://bg.dbpedia.org/resource/
	d.sl   http://sl.dbpedia.org/resource/
	d.eu   http://eu.dbpedia.org/resource/
	d.war   http://war.dbpedia.org/resource/
	d.et   http://et.dbpedia.org/resource/
	d.hr   http://hr.dbpedia.org/resource/
	d.ms   http://ms.dbpedia.org/resource/
	d.hi   http://hi.dbpedia.org/resource/
	d.sh   http://sh.dbpedia.org/resource/

	dwp.de   http://de.dbpedia.org/property/wikiPage
	dwp.fr   http://fr.dbpedia.org/property/wikiPage
	dwp.en   http://en.dbpedia.org/property/wikiPage
	dwp.es   http://es.dbpedia.org/property/wikiPage
	dwp.it   http://it.dbpedia.org/property/wikiPage
	dwp.nl   http://nl.dbpedia.org/property/wikiPage
	dwp.ru   http://ru.dbpedia.org/property/wikiPage
	dwp.sv   http://sv.dbpedia.org/property/wikiPage
	dwp.pl   http://pl.dbpedia.org/property/wikiPage
	dwp.ja   http://ja.dbpedia.org/property/wikiPage
	dwp.pt   http://pt.dbpedia.org/property/wikiPage
	dwp.ar   http://ar.dbpedia.org/property/wikiPage
	dwp.zh   http://zh.dbpedia.org/property/wikiPage
	dwp.uk   http://uk.dbpedia.org/property/wikiPage
	dwp.ca   http://ca.dbpedia.org/property/wikiPage
	dwp.no   http://no.dbpedia.org/property/wikiPage
	dwp.fi   http://fi.dbpedia.org/property/wikiPage
	dwp.cs   http://cs.dbpedia.org/property/wikiPage
	dwp.hu   http://hu.dbpedia.org/property/wikiPage
	dwp.tr   http://tr.dbpedia.org/property/wikiPage
	dwp.ro   http://ro.dbpedia.org/property/wikiPage
	dwp.sw   http://sw.dbpedia.org/property/wikiPage
	dwp.ko   http://ko.dbpedia.org/property/wikiPage
	dwp.kk   http://kk.dbpedia.org/property/wikiPage
	dwp.vi   http://vi.dbpedia.org/property/wikiPage
	dwp.da   http://da.dbpedia.org/property/wikiPage
	dwp.eo   http://eo.dbpedia.org/property/wikiPage
	dwp.sr   http://sr.dbpedia.org/property/wikiPage
	dwp.id   http://id.dbpedia.org/property/wikiPage
	dwp.lt   http://lt.dbpedia.org/property/wikiPage
	dwp.vo   http://vo.dbpedia.org/property/wikiPage
	dwp.sk   http://sk.dbpedia.org/property/wikiPage
	dwp.he   http://he.dbpedia.org/property/wikiPage
	dwp.fa   http://fa.dbpedia.org/property/wikiPage
	dwp.bg   http://bg.dbpedia.org/property/wikiPage
	dwp.sl   http://sl.dbpedia.org/property/wikiPage
	dwp.eu   http://eu.dbpedia.org/property/wikiPage
	dwp.war   http://war.dbpedia.org/property/wikiPage
	dwp.et   http://et.dbpedia.org/property/wikiPage
	dwp.hr   http://hr.dbpedia.org/property/wikiPage
	dwp.ms   http://ms.dbpedia.org/property/wikiPage
	dwp.hi   http://hi.dbpedia.org/property/wikiPage
	dwp.sh   http://sh.dbpedia.org/property/wikiPage

	dp.de   http://de.dbpedia.org/property/
	dp.fr   http://fr.dbpedia.org/property/
	dp.en   http://en.dbpedia.org/property/
	dp.es   http://es.dbpedia.org/property/
	dp.it   http://it.dbpedia.org/property/
	dp.nl   http://nl.dbpedia.org/property/
	dp.ru   http://ru.dbpedia.org/property/
	dp.sv   http://sv.dbpedia.org/property/
	dp.pl   http://pl.dbpedia.org/property/
	dp.ja   http://ja.dbpedia.org/property/
	dp.pt   http://pt.dbpedia.org/property/
	dp.ar   http://ar.dbpedia.org/property/
	dp.zh   http://zh.dbpedia.org/property/
	dp.uk   http://uk.dbpedia.org/property/
	dp.ca   http://ca.dbpedia.org/property/
	dp.no   http://no.dbpedia.org/property/
	dp.fi   http://fi.dbpedia.org/property/
	dp.cs   http://cs.dbpedia.org/property/
	dp.hu   http://hu.dbpedia.org/property/
	dp.tr   http://tr.dbpedia.org/property/
	dp.ro   http://ro.dbpedia.org/property/
	dp.sw   http://sw.dbpedia.org/property/
	dp.ko   http://ko.dbpedia.org/property/
	dp.kk   http://kk.dbpedia.org/property/
	dp.vi   http://vi.dbpedia.org/property/
	dp.da   http://da.dbpedia.org/property/
	dp.eo   http://eo.dbpedia.org/property/
	dp.sr   http://sr.dbpedia.org/property/
	dp.id   http://id.dbpedia.org/property/
	dp.lt   http://lt.dbpedia.org/property/
	dp.vo   http://vo.dbpedia.org/property/
	dp.sk   http://sk.dbpedia.org/property/
	dp.he   http://he.dbpedia.org/property/
	dp.fa   http://fa.dbpedia.org/property/
	dp.bg   http://bg.dbpedia.org/property/
	dp.sl   http://sl.dbpedia.org/property/
	dp.eu   http://eu.dbpedia.org/property/
	dp.war   http://war.dbpedia.org/property/
	dp.et   http://et.dbpedia.org/property/
	dp.hr   http://hr.dbpedia.org/property/
	dp.ms   http://ms.dbpedia.org/property/
	dp.hi   http://hi.dbpedia.org/property/
	dp.sh   http://sh.dbpedia.org/property/

	address http://schemas.talis.com/2005/address/schema#
	admin   http://webns.net/mvcb/
	atom    http://atomowl.org/ontologies/atomrdf#
	atom   http://www.w3.org/2005/Atom
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
	ff  http://api.friendfeed.com/2008/03
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
