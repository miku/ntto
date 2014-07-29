ntto
====

Shrink N-Triples by applying namespace abbreviations.

[![Build Status](http://img.shields.io/travis/miku/ntto.svg?style=flat)](https://travis-ci.org/miku/ntto)

Mode of operation
-----------------

ntto takes a RULES file (alternatively uses some builtin-rules) to abbreviate
common prefixes in n-triple files.

ntto does not do the replacements itself, but outsources it to a couple of
[sed](http://en.wikipedia.org/wiki/Sed) processes which will be run in parallel.

This will shrink in the order of 30k to 50k lines per second. The resulting
files can be up to 50% of the size of the original file.

Example rules file
------------------

    $ cat RULES
    # example rules file
    dbp             http://dbpedia.org/resource/
    gnd             http://d-nb.info/gnd/
    dnbes           http://d-nb.info/standards/elementset/gnd#
    dnbac           http://d-nb.info/standards/vocab/gnd/geographic-area-code#
    dnbv            http://d-nb.info/standards/vocab/gnd/

    viaf            http://viaf.org/viaf/
    frbr            http://rdvocab.info/uri/schema/FRBRentitiesRDA/
    rdgr            http://rdvocab.info/ElementsGr2/

    # empty lines are ignored, as are comments

    foaf            http://xmlns.com/foaf/0.1/
    rdf             http://www.w3.org/1999/02/22-rdf-syntax-ns#
    rdfs            http://www.w3.org/2000/01/rdf-schema#
    schema          http://schema.org/
    dc              http://purl.org/dc/elements/1.1/
    dcterms         http://purl.org/dc/terms/

Usage
-----

    $ ntto
    Usage: ntto [OPTIONS] FILE
      -c=false: dump constructed sed command and exit
      -cpuprofile="": write cpu profile to file
      -d=false: dump rules and exit
      -n="<NULL>": string to indicate empty string replacement
      -o="": output file to write result to
      -r="": path to rules file, use built-in if none given
      -v=false: prints current version and exits
      -w=4: number of sed processes

Example conversion
------------------

Before:

    <http://d-nb.info/gnd/1-2> <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://d-nb.info/standards/elementset/gnd#SeriesOfConferenceOrEvent> .
    <http://d-nb.info/gnd/1-2> <http://d-nb.info/standards/elementset/gnd#gndIdentifier> "1-2" .
    <http://d-nb.info/gnd/1-2> <http://d-nb.info/standards/elementset/gnd#oldAuthorityNumber> "(DE-588b)1-2" .
    <http://d-nb.info/gnd/1-2> <http://d-nb.info/standards/elementset/gnd#variantNameForTheConferenceOrEvent> "Conferentie van Niet-Kernwapenstaten" .
    <http://d-nb.info/gnd/1-2> <http://d-nb.info/standards/elementset/gnd#variantNameForTheConferenceOrEvent> "Conference on Non-Nuclear Weapon States" .
    <http://d-nb.info/gnd/1-2> <http://d-nb.info/standards/elementset/gnd#preferredNameForTheConferenceOrEvent> "Conference of Non-Nuclear Weapon States" .
    <http://d-nb.info/gnd/2-4> <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://d-nb.info/standards/elementset/gnd#ConferenceOrEvent> .
    <http://d-nb.info/gnd/2-4> <http://d-nb.info/standards/elementset/gnd#gndIdentifier> "2-4" .
    <http://d-nb.info/gnd/2-4> <http://d-nb.info/standards/elementset/gnd#oldAuthorityNumber> "(DE-588b)2-4" .
    ...

After:

    <gnd:1-2> <rdf:type> <dnbes:SeriesOfConferenceOrEvent> .
    <gnd:1-2> <dnbes:gndIdentifier> "1-2" .
    <gnd:1-2> <dnbes:oldAuthorityNumber> "(DE-588b)1-2" .
    <gnd:1-2> <dnbes:variantNameForTheConferenceOrEvent> "Conferentie van Niet-Kernwapenstaten" .
    <gnd:1-2> <dnbes:variantNameForTheConferenceOrEvent> "Conference on Non-Nuclear Weapon States" .
    <gnd:1-2> <dnbes:preferredNameForTheConferenceOrEvent> "Conference of Non-Nuclear Weapon States" .
    <gnd:2-4> <rdf:type> <dnbes:ConferenceOrEvent> .
    <gnd:2-4> <dnbes:gndIdentifier> "2-4" .
    <gnd:2-4> <dnbes:oldAuthorityNumber> "(DE-588b)2-4" .
    ...
