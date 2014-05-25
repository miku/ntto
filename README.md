README
======

`nttoldj` converts N-Triples into Line Delimited JSON.

The MIME-Type for Line Delimited JSON is currently `application/x-ldjson`.
Suggested file extensions are `.ldj` and `.ldjson`.

[![Gobuild download](http://gobuild.io/badge/github.com/miku/nttoldj/download.png)](http://gobuild.io/download/github.com/miku/nttoldj)

Usage
-----

    Usage of ./nttoldj [OPTIONS] FILE
      -a=false: abbreviate triples
      -f="json": output format (json, xml, tsv)
      -p=false: cpu profile

Prefix rewriting
----------------

One use case for LDJ formatted N-Triples is the need to index or store
large amount of triples. Storing common prefixes over and over again
is redundant. Rewriting prefixes can save huge amounts of space and increase
the overall performance of the data store.

There are over hundred rewrite rules (hardcoded):

    dbp http://dbpedia.org/resource/
    dbpopp http://dbpedia.org/ontology/PopulatedPlace/
    dbpo http://dbpedia.org/ontology/
    dbpp http://dbpedia.org/property/
    foaf    http://xmlns.com/foaf/0.1/
    rdf http://www.w3.org/1999/02/22-rdf-syntax-ns#
    rdfa    http://www.w3.org/ns/rdfa#
    rdfdf   http://www.openlinksw.com/virtrdf-data-formats#
    rdfs    http://www.w3.org/2000/01/rdf-schema#
    dc  http://purl.org/dc/elements/1.1/
    dcterms http://purl.org/dc/terms/
    umbel   http://umbel.org/umbel#
    umbel-ac    http://umbel.org/umbel/ac/
    umbel-sc    http://umbel.org/umbel/sc/
    ...

Performance
-----------

    $ # i5/SDD
    $ wc -l labels_en.nt
    10141501
    $ time go run nttoldj.go -a labels_en.nt > labels_en.json
    4 workers/126 rules

    real    1m43.030s
    user    5m3.715s
    sys     0m57.915s
