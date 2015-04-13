ntto
====

Minimal n-triples toolkit. It can:

* shrink n-triples by applying namespace abbreviations (given some rules)
* convert n-triples to line delimited JSON (.ldj)

[![Build Status](http://img.shields.io/travis/miku/ntto.svg?style=flat)](https://travis-ci.org/miku/ntto)

To list the abbreviation rules, run:

    $ ntto -d

To create an abbreviated NT file from an NT file, run:

    $ ntto -o OUTPUT.NT -a FILE.nt

To create an abbreviated JSON file from an NT file, run:

    $ ntto -a -j FILE.nt > OUTPUT.LDJ

To create an abbreviated JSON file from an NT file while ignoring conversion errors, run:

    $ ntto -a -j -i FILE.nt > OUTPUT.LDJ

To create an abbreviated JSON file from an NT file while ignoring conversion errors and using a custom RULES file, run:

    $ ntto -r RULES -a -j -i FILE.nt > OUTPUT.LDJ

Installation
------------

RPM and DEB packages can be found under [releases](https://github.com/miku/ntto/releases).

With a proper Go setup, a

    $ go get github.com/miku/ntto/cmd/ntto

should work as well.

Usage
-----

    $ ntto
    Usage: ntto [OPTIONS] FILE
      -a=false: abbreviate n-triples using rules
      -c=false: dump constructed sed command and exit
      -cpuprofile="": write cpu profile to file
      -d=false: dump rules and exit
      -i=false: ignore conversion errors
      -j=false: convert nt to json
      -n="<NULL>": string to indicate empty string replacement
      -o="": output file to write result to
      -r="": path to rules file, use built-in if none given
      -v=false: prints current version and exits
      -w=4: parallelism measure

Mode of operation
-----------------

`ntto` takes a RULES file (alternatively uses some [hardwired](https://github.com/miku/ntto/blob/master/rules.go) rules) to abbreviate
common prefixes in a n-triple file. `ntto` does not do the replacements itself, but outsources it to external programs, like `replace` or `perl`.

With the help of `replace` ntto can shorten up to 3M lines per second. The resulting
file size can be up to 50% of the size of the original file.

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

Performance data point
----------------------

    $ wc -l file.nt
    114171541

    $ time ntto -o output.nt -a file.nt
    real    1m51.202s
    user    1m3.626s
    sys     0m13.602s

    $ time ntto -a -j file.nt > output.ldj
    real    15m47.872s
    user    16m19.516s
    sys      2m3.013s

Sometimes, less is more, but YMMV:

    $ time ntto -w 2 -a -j file.nt > output.ldj
    real    12m3.619s
    user    15m17.422s
    sys     2m14.430s
