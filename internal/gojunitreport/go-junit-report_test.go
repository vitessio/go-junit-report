package gojunitreport

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

const testDataDir = "../../testdata/"

var matchTest = flag.String("match", "", "only test testdata matching this pattern")

var testConfigs = map[int]Config{
	1: {ExitCode: 1},
	5: {SkipXMLHeader: true},
	6: {SkipXMLHeader: true},
	7: {PackageName: "test/package"},
	8: {ExitCode: 1},
	12: {ExitCode: 1},
	17: {ExitCode: 1},
	30: {ExitCode: 1},
	36: {ExitCode: 1},
	37: {ExitCode: 1},
	101: {ExitCode: 1},
	103: {ExitCode: 1},
	104: {ExitCode: 1},
	110: {ExitCode: 1},
	112: {ExitCode: 1},
}

func TestRun(t *testing.T) {
	matchRegex := compileMatch(t)

	files, err := filepath.Glob(testDataDir + "*.txt")
	if err != nil {
		t.Fatalf("error finding files in testdata: %v", err)
	}

	for _, file := range files {
		if !matchRegex.MatchString(file) {
			continue
		}

		conf, reportFile, err := testFileConfig(strings.TrimPrefix(file, testDataDir))
		if err != nil {
			t.Errorf("testFileConfig error: %v", err)
			continue
		}

		t.Run(filepath.Base(file), func(t *testing.T) {
			testRun(file, testDataDir+reportFile, conf, t)
		})
	}
}

func testRun(inputFile, reportFile string, config Config, t *testing.T) {
	input, err := os.Open(inputFile)
	if err != nil {
		t.Fatalf("error opening input file: %v", err)
	}
	defer input.Close()

	wantReport, err := ioutil.ReadFile(reportFile)
	if os.IsNotExist(err) {
		t.Skipf("Skipping test with missing report file: %s", reportFile)
	} else if err != nil {
		t.Fatalf("error loading report file: %v", err)
	}

	config.Parser = "gotest"
	if strings.HasSuffix(inputFile, ".gojson.txt") {
		config.Parser = "gojson"
	}
	config.Hostname = "hostname"
	config.Properties = map[string]string{"go.version": "1.0"}
	config.TimestampFunc = func() time.Time {
		return time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	var output bytes.Buffer
	r, err := config.Run(input, &output)
	if err != nil {
		t.Fatal(err)
	}
	if config.ExitCode > 0 && r.Failures() < 1 {
		t.Errorf("Unexpected exit code want: 1, got: 0")
	}

	if diff := cmp.Diff(string(wantReport), output.String()); diff != "" {
		t.Errorf("Unexpected report diff (-want, +got):\n%v", diff)
	}
}

func testFileConfig(filename string) (config Config, reportFile string, err error) {
	var prefix string
	if idx := strings.IndexByte(filename, '-'); idx < 0 {
		return config, "", fmt.Errorf("testdata file does not contain a dash (-); expected name `{id}-{name}.txt` got `%s`", filename)
	} else {
		prefix = filename[:idx]
	}
	id, err := strconv.Atoi(prefix)
	if err != nil {
		return config, "", fmt.Errorf("testdata file did not start with a valid number: %w", err)
	}
	return testConfigs[id], fmt.Sprintf("%s-report.xml", prefix), nil
}

func compileMatch(t *testing.T) *regexp.Regexp {
	rx, err := regexp.Compile(*matchTest)
	if err != nil {
		t.Fatalf("Error compiling -match flag %q: %v", *matchTest, err)
	}
	return rx
}
