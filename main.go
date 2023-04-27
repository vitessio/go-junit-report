package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vitessio/go-junit-report/internal/gojunitreport"
	"github.com/vitessio/go-junit-report/parser/gotest"
)

var (
	Version   = "v2.0.0-dev"
	Revision  = "HEAD"
	BuildTime string
)

var (
	noXMLHeader = flag.Bool("no-xml-header", false, "do not print xml header")
	packageName = flag.String("package-name", "", "specify a default package `name` to use if output does not contain a package name")
	setExitCode = flag.Bool("set-exit-code", false, "set exit code to 1 if tests failed")
	version     = flag.Bool("version", false, "print version")
	input       = flag.String("in", "", "read go test log from `file`")
	output      = flag.String("out", "", "write XML report to `file`")
	iocopy      = flag.Bool("iocopy", false, "copy input to stdout; can only be used in conjunction with -out")
	properties  = make(keyValueFlag)
	parser      = flag.String("parser", "gotest", "set input parser: gotest, gojson")
	mode        = flag.String("subtest-mode", "", "set subtest `mode`: ignore-parent-results (subtest parents always pass), exclude-parents (subtest parents are excluded from the report)")

	// debug flags
	printEvents = flag.Bool("debug.print-events", false, "print events generated by the go test parser")

	// deprecated flags
	goVersionFlag = flag.String("go-version", "", "(deprecated, use -prop) the value to use for the go.version property in the generated XML")
)

func main() {
	flag.Var(&properties, "p", "add `key=value` property to generated report; repeat this flag to add multiple properties.")
	flag.Parse()

	if *iocopy && *output == "" {
		exitf("you must specify an output file with -out when using -iocopy")
	}

	if *version {
		fmt.Printf("go-junit-report %s %s (%s)\n", Version, BuildTime, Revision)
		return
	}

	if *goVersionFlag != "" {
		fmt.Fprintf(os.Stderr, "the -go-version flag is deprecated and will be removed in the future, use the -p flag instead.\n")
		properties["go.version"] = *goVersionFlag
	}

	subtestMode := gotest.SubtestModeDefault
	if *mode != "" {
		var err error
		if subtestMode, err = gotest.ParseSubtestMode(*mode); err != nil {
			exitf("invalid value for -subtest-mode: %s\n", err)
		}
	}

	if flag.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "invalid argument(s): %s\n", strings.Join(flag.Args(), " "))
		fmt.Fprintf(os.Stderr, "%s does not accept positional arguments\n", os.Args[0])
		flag.Usage()
		exitf("")
	}

	var in io.Reader = os.Stdin
	if *input != "" {
		f, err := os.Open(*input)
		if err != nil {
			exitf("error opening input file: %v", err)
		}
		defer f.Close()
		in = f
	}

	var out io.Writer = os.Stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			exitf("error creating output file: %v", err)
		}
		defer f.Close()
		out = f
	}

	if *iocopy {
		in = io.TeeReader(in, os.Stdout)
	}

	hostname, _ := os.Hostname() // ignore error

	config := gojunitreport.Config{
		Parser:        *parser,
		Hostname:      hostname,
		PackageName:   *packageName,
		SkipXMLHeader: *noXMLHeader,
		SubtestMode:   subtestMode,
		Properties:    properties,
		PrintEvents:   *printEvents,
	}
	report, err := config.Run(in, out)
	if err != nil {
		exitf("error: %v\n", err)
	}

	if *setExitCode && report.Failures() > 0 {
		os.Exit(1)
	}
}

func exitf(msg string, args ...interface{}) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, msg+"\n", args...)
	}
	os.Exit(2)
}

type keyValueFlag map[string]string

func (f *keyValueFlag) String() string {
	if f != nil {
		var pairs []string
		for k, v := range *f {
			pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
		}
		return strings.Join(pairs, ",")
	}
	return ""
}

func (f *keyValueFlag) Set(value string) error {
	idx := strings.IndexByte(value, '=')
	if idx == -1 {
		return fmt.Errorf("%v is not specified as \"key=value\"", value)
	}
	k, v := value[:idx], value[idx+1:]
	(*f)[k] = v
	return nil
}
