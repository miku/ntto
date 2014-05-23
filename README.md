README
======

`nttoldj` converts N-Triples into Line Delimited JSON.

The MIME-Type for Line Delimited JSON is currently `application/x-ldjson`. Suggested
file extensions are `.ldj` and `.ldjson`.

Usage
-----

    $ nttoldj FILE


Prefix rewriting
----------------

One use case for LDJ formatted N-Triples is the need to index or store
large amount of triples. Storing common prefixes over and over again
is redundant. Rewriting prefixes can save huge amounts of space and increase
the overall performance of the data store.

There are a couple of supplied rewrite rules:

    http://viaf.org/viaf/([0-9]+)                          viaf:$1
    http://d-nb.info/gnd/([0-9X-]+)                        gnd:$1
    http://xmlns.com/foaf/0.1/([^/]+)                      foaf:$1
    http://rdvocab.info/uri/schema/FRBRentitiesRDA/([^/]+) frbr:$1

Example savings for 100 million `frbr:...` records:

* with full url: 4.3GB
* with short prefix: 381M
