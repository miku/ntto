package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/miku/ntto"
)

func Worker(queue chan *string, out chan *ntto.Triple, wg *sync.WaitGroup, ignore *bool) {
	defer wg.Done()
	for b := range queue {
		triple, err := ntto.ParseNTriple(*b)
		if err != nil {
			if !*ignore {
				log.Fatalln(err)
			} else {
				log.Println(err)
			}
		}
		out <- triple
	}
}

func Marshaller(writer io.Writer, in chan *ntto.Triple, done chan bool, ignore *bool) {
	for triple := range in {
		b, err := json.Marshal(triple)
		if err != nil {
			if !*ignore {
				log.Fatalln(err)
			} else {
				log.Println(err)
			}
		}
		writer.Write(b)
		writer.Write([]byte("\n"))
	}
	done <- true
}

func main() {

	executive := "replace"
	_, err := exec.LookPath("replace")
	if err != nil {
		executive = "perl"
	}

	_, err = exec.LookPath("perl")
	if err != nil {
		log.Fatalln("This program requires perl or replace.")
		os.Exit(1)
	}

	abbreviate := flag.Bool("a", false, "abbreviate n-triples using rules")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	dumpCommand := flag.Bool("c", false, "dump constructed sed command and exit")
	dumpRules := flag.Bool("d", false, "dump rules and exit")
	ignore := flag.Bool("i", false, "ignore conversion errors")
	jsonOutput := flag.Bool("j", false, "convert nt to json")
	nullValue := flag.String("n", "<NULL>", "string to indicate empty string replacement")
	outFile := flag.String("o", "", "output file to write result to")
	rulesFile := flag.String("r", "", "path to rules file, use built-in if none given")
	version := flag.Bool("v", false, "prints current version and exits")
	numWorkers := flag.Int("w", runtime.NumCPU(), "parallelism measure")

	flag.Parse()

	runtime.GOMAXPROCS(*numWorkers)

	var PrintUsage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] FILE\n", os.Args[0])
		flag.PrintDefaults()
	}

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

	var rules []ntto.Rule

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

	if flag.NArg() < 1 {
		PrintUsage()
		os.Exit(1)
	}

	filename := flag.Args()[0]
	var output string

	if *abbreviate {
		if *outFile == "" {
			tmp, err := ioutil.TempFile("", "ntto-")
			output = tmp.Name()
			log.Printf("No explicit [-o]utput given, writing to %s\n", output)
			if err != nil {
				log.Fatalln(err)
			}
		} else {
			output = *outFile
		}

		var command string
		if executive == "perl" {
			command = fmt.Sprintf("%s > %s", ntto.SedifyNull(rules, *numWorkers, filename, *nullValue), output)
		} else {
			command = fmt.Sprintf("%s > %s", ntto.ReplacifyNull(rules, filename, *nullValue), output)
		}
		if *dumpCommand {
			fmt.Println(command)
			os.Exit(0)
		}
		_, err = exec.Command("sh", "-c", command).Output()
		if err != nil {
			log.Fatalln(err)
		}
		// set filename to abbreviated output, so we can use combine -j -a
		filename = output
	}

	if *jsonOutput {
		var file *os.File
		if filename == "-" {
			file = os.Stdin
		} else {
			file, err = os.Open(filename)
			defer file.Close()
			if err != nil {
				log.Fatalln(err)
			}
		}

		queue := make(chan *string)
		results := make(chan *ntto.Triple)
		done := make(chan bool)

		writer := bufio.NewWriter(os.Stdout)
		defer writer.Flush()
		go Marshaller(writer, results, done, ignore)

		var wg sync.WaitGroup
		for i := 0; i < *numWorkers; i++ {
			wg.Add(1)
			go Worker(queue, results, &wg, ignore)
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
		close(queue)
		wg.Wait()
		close(results)
		select {
		case <-time.After(1e9):
			break
		case <-done:
			break
		}
		// remove abbreviated tempfile output, if possible
		if *outFile == "" {
			_ = os.Remove(output)
		}
	}
}
