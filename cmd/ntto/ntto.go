package main

import (
	"flag"
	"fmt"
	"github.com/miku/ntto"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
)

type Work struct {
	Line             *string
	Abbreviate       *bool
	LanguageLiterals *[]string
	OutputFormat     *string
	Rules            *[]ntto.Rule
}

func main() {

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	version := flag.Bool("v", false, "prints current version and exits")
	dumpRules := flag.Bool("d", false, "dump rules and exit")
	workers := flag.Int("w", runtime.NumCPU(), "number of workers")
	rulesFile := flag.String("r", "", "path to rules file, use built-in if none given")

	flag.Parse()

	var PrintUsage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] FILE\n", os.Args[0])
		flag.PrintDefaults()
	}

	runtime.GOMAXPROCS(*workers)

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatalln(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if *version {
		fmt.Println(ntto.AppVersion)
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		PrintUsage()
		os.Exit(1)
	}

	var rules []ntto.Rule
	var err error

	if *rulesFile == "" {
		rules, err = ntto.ParseRules(ntto.DefaultRules)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		b, err := ioutil.ReadFile(*rulesFile)
		if err != nil {
			log.Fatalln(err)
		}
		rules, err = ntto.ParseRules(string(b))
		if err != nil {
			log.Fatalln(err)
		}
	}

	if *dumpRules {
		fmt.Println(ntto.DumpRules(rules))
		os.Exit(0)
	}

	file, err := os.Open(flag.Args()[0])
	if err != nil {
		log.Fatalln(err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			log.Fatalln(err)
		}
	}()

}
