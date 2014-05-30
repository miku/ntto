README
======

`nttoldj` converts N-Triples into Line Delimited JSON and optionally applies prefix abbreviations.

The MIME-Type for Line Delimited JSON is currently `application/x-ldjson`.
Suggested file extensions are `.ldj` and `.ldjson`.

[![Gobuild download](http://gobuild.io/badge/github.com/miku/nttoldj/download.png)](http://gobuild.io/download/github.com/miku/nttoldj)

Usage
-----

    Usage of ./nttoldj [OPTIONS] FILE
      -a=false: abbreviate triples
      -d=false: dump rules as TSV to stdout
      -f="json": output format (json, xml, tsv)
      -l="": only keep literals of the given language
      -p=false: cpu profile
      -v=false: prints current version and exits



Prefix rewriting
----------------

One use case for LDJ formatted N-Triples is the need to index or store
large amount of triples. Storing common prefixes over and over again
is redundant. Rewriting prefixes can save huge amounts of space and increase
the overall performance of the data store.

There are a few hundred rewrite rules (hardcoded for now):

    d        http://dbpedia.org/resource/
    # dbpedia languages (more below)
    d.de     http://de.dbpedia.org/resource/
    do.pp    http://dbpedia.org/ontology/PopulatedPlace/
    do.wp    http://dbpedia.org/ontology/wikiPage
    do       http://dbpedia.org/ontology/
    dp.wp    http://dbpedia.org/property/wikiPage
    dp       http://dbpedia.org/property/

    gnd      http://d-nb.info/gnd/
    dnb.es   http://d-nb.info/standards/elementset/gnd#
    dnb.area http://d-nb.info/standards/vocab/gnd/geographic-area-code#
    dnb.voc  http://d-nb.info/standards/vocab/gnd/

    viaf     http://viaf.org/viaf/

    foaf     http://xmlns.com/foaf/0.1/
    rdf      http://www.w3.org/1999/02/22-rdf-syntax-ns#
    rdfs     http://www.w3.org/2000/01/rdf-schema#
    schema   http://schema.org/
    dc       http://purl.org/dc/elements/1.1/
    dcterms  http://purl.org/dc/terms/
    ...


Performance
-----------

    $ # i5/SDD
    $ wc -l labels_en.nt
    10141501

    $ time go run nttoldj.go -a labels_en.nt > labels_en.ldj
    4 workers/126 rules

    real    1m43.030s
    user    5m3.715s
    sys     0m57.915s

----

    $ # i5/HDD
    $ wc -l page_links_en.nt
    172308906

    $ time nttoldj -a page_links_en.nt > page_links_en.ldj
    4 workers/127 rules

    real    23m37.956s
    user    51m56.732s
    sys     6m0.088s

----

    $ # Xeon/HDD
    $ wc -l page_links_en.nt
    172308906

    $ time nttoldj -a page_links_en.nt > page_links_en.ldj
    8 workers/127 rules

    real    27m40.641s
    user    74m51.975s
    sys     20m14.763s

Converting about 500M triples with diverse prefixes takes less than two hours.

