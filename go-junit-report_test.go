package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jstemmer/go-junit-report/v2/pkg/gtr"
	"github.com/jstemmer/go-junit-report/v2/pkg/junit"
	"github.com/jstemmer/go-junit-report/v2/pkg/parser/gotest"

	"github.com/google/go-cmp/cmp"
)

var matchTest = flag.String("match", "", "only test testdata matching this pattern")

type TestCase struct {
	name        string
	reportName  string
	noXMLHeader bool
	packageName string
}

var testCases = []TestCase{
	{
		name:       "01-pass.txt",
		reportName: "01-report.xml",
	},
	{
		name:       "02-fail.txt",
		reportName: "02-report.xml",
	},
	{
		name:       "03-skip.txt",
		reportName: "03-report.xml",
	},
	{
		name:       "04-go_1_4.txt",
		reportName: "04-report.xml",
	},
	{
		name:        "05-no_xml_header.txt",
		reportName:  "05-report.xml",
		noXMLHeader: true,
	},
	{
		name:        "06-mixed.txt",
		reportName:  "06-report.xml",
		noXMLHeader: true,
	},
	{
		name:        "07-compiled_test.txt",
		reportName:  "07-report.xml",
		packageName: "test/package",
	},
	{
		name:       "08-parallel.txt",
		reportName: "08-report.xml",
	},
	{
		name:       "09-coverage.txt",
		reportName: "09-report.xml",
	},
	{
		name:       "10-multipkg-coverage.txt",
		reportName: "10-report.xml",
	},
	{
		name:       "11-go_1_5.txt",
		reportName: "11-report.xml",
	},
	{
		name:       "12-go_1_7.txt",
		reportName: "12-report.xml",
	},
	{
		name:       "13-syntax-error.txt",
		reportName: "13-report.xml",
	},
	{
		name:       "14-panic.txt",
		reportName: "14-report.xml",
	},
	{
		name:       "15-empty.txt",
		reportName: "15-report.xml",
	},
	{
		name:       "16-repeated-names.txt",
		reportName: "16-report.xml",
	},
	{
		name:       "17-race.txt",
		reportName: "17-report.xml",
	},
	{
		name:       "18-coverpkg.txt",
		reportName: "18-report.xml",
	},
	{
		name:       "19-pass.txt",
		reportName: "19-report.xml",
	},
	{
		name:       "20-parallel.txt",
		reportName: "20-report.xml",
	},
	{
		name:       "21-cached.txt",
		reportName: "21-report.xml",
	},
	{
		name:       "22-bench.txt",
		reportName: "22-report.xml",
	},
	{
		name:       "23-benchmem.txt",
		reportName: "23-report.xml",
	},
	{
		name:       "24-benchtests.txt",
		reportName: "24-report.xml",
	},
	{
		name:       "25-benchcount.txt",
		reportName: "25-report.xml",
	},
	{
		name:       "26-testbenchmultiple.txt",
		reportName: "26-report.xml",
	},
	{
		name:       "27-benchdecimal.txt",
		reportName: "27-report.xml",
	},
	{
		name:       "28-bench-1cpu.txt",
		reportName: "28-report.xml",
	},
	{
		name:       "29-bench-16cpu.txt",
		reportName: "29-report.xml",
	},
	{
		// generated by running go test on https://gist.github.com/liggitt/09a021ccec988b19917e0c2d60a18ee9
		name:       "30-stdout.txt",
		reportName: "30-report.xml",
	},
	{
		name:       "31-syntax-error-test-binary.txt",
		reportName: "31-report.xml",
	},
	{
		name:       "32-failed-summary.txt",
		reportName: "32-report.xml",
	},
	{
		name:       "33-bench-mb.txt",
		reportName: "33-report.xml",
	},
	{
		name:       "34-notest.txt",
		reportName: "34-report.xml",
	},
}

func TestNewOutput(t *testing.T) {
	matchRegex := compileMatch(t)
	for _, testCase := range testCases {
		if !matchRegex.MatchString(testCase.name) {
			continue
		}

		t.Run(testCase.name, func(t *testing.T) {
			testReport(testCase.name, testCase.reportName, testCase.packageName, t)
		})
	}
}

func testReport(input, reportFile, packageName string, t *testing.T) {
	file, err := os.Open("testdata/" + input)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	events, err := gotest.Parse(file)
	if err != nil {
		t.Fatal(err)
	}

	if *printEvents {
		for _, event := range events {
			t.Logf("Event: %+v", event)
		}
	}

	testTime := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	actual := gtr.JUnit(gtr.FromEvents(events, packageName), "hostname", testTime)
	// Remove any new properties for backwards compatibility
	actual = dropNewProperties(actual)

	expectedXML, err := loadTestReport(reportFile, "")
	if err != nil {
		t.Fatal(err)
	}

	var expected junit.Testsuites
	if err := xml.Unmarshal([]byte(expectedXML), &expected); err != nil {
		t.Fatal(err)
	}

	expected = modifyForBackwardsCompat(expected)
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("Unexpected report diff (-got, +want):\n%v", diff)
	}
}

func modifyForBackwardsCompat(testsuites junit.Testsuites) junit.Testsuites {
	testsuites.XMLName.Local = ""
	for i, suite := range testsuites.Suites {
		for j := range suite.Testcases {
			testsuites.Suites[i].Testcases[j].Classname = suite.Name
		}

		if suite.Properties != nil {
			if covIdx, covProp := getProperty("coverage.statements.pct", *suite.Properties); covIdx > -1 {
				pct, _ := strconv.ParseFloat(covProp.Value, 64)
				(*testsuites.Suites[i].Properties)[covIdx].Value = fmt.Sprintf("%.2f", pct)
			}
			testsuites.Suites[i].Properties = dropProperty("go.version", suite.Properties)
		}
	}
	return testsuites
}

func dropNewProperties(testsuites junit.Testsuites) junit.Testsuites {
	for i, suite := range testsuites.Suites {
		if suite.Properties == nil {
			continue
		}
		ps := suite.Properties
		ps = dropProperty("goos", ps)
		ps = dropProperty("goarch", ps)
		ps = dropProperty("pkg", ps)
		testsuites.Suites[i].Properties = ps
	}
	return testsuites
}

func dropProperty(name string, properties *[]junit.Property) *[]junit.Property {
	if properties == nil {
		return nil
	}
	var props []junit.Property
	for _, prop := range *properties {
		if prop.Name != name {
			props = append(props, prop)
		}
	}
	if len(props) == 0 {
		return nil
	}
	return &props
}

func getProperty(name string, properties []junit.Property) (int, junit.Property) {
	for i, prop := range properties {
		if prop.Name == name {
			return i, prop
		}
	}
	return -1, junit.Property{}
}

func loadTestReport(name, goVersion string) (string, error) {
	contents, err := ioutil.ReadFile("testdata/" + name)
	if err != nil {
		return "", err
	}

	if goVersion == "" {
		// if goVersion is not specified, default to runtime version
		goVersion = runtime.Version()
	}

	// replace value="1.0" With actual version
	report := strings.Replace(string(contents), `value="1.0"`, fmt.Sprintf(`value="%s"`, goVersion), -1)

	return report, nil
}

func compileMatch(t *testing.T) *regexp.Regexp {
	rx, err := regexp.Compile(*matchTest)
	if err != nil {
		t.Fatalf("Error compiling -match flag %q: %v", *matchTest, err)
	}
	return rx
}
