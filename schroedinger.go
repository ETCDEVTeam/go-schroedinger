package schroedinger

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const commentPattern = "#"

var errCommentLine = errors.New("comment line")
var errEmptyLine = errors.New("empty line")

var trialsAllowed int

// different for windows
var goExecutablePath string
var commandPrefix []string

type test struct {
	pkg    string
	name   string
	trials int
}

func (t *test) String() string {
	return fmt.Sprintf("%s %s", t.pkg, t.name)
}

func init() {
	goExecutablePath = getGoPath()
	commandPrefix = getCommandPrefix()
}

func getGoPath() string {
	return filepath.Join(runtime.GOROOT(), "bin", "go")
}

func getCommandPrefix() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/C"}
	}
	return []string{"/bin/sh", "-c"}
}

func parseLinePackageTest(s string) *test {
	t := &test{}
	lsep := strings.Split(s, " ")
	t.pkg = lsep[0]
	if len(lsep) > 1 {
		t.name = lsep[1]
	}
	t.pkg = strings.Replace(t.pkg, "/", string(filepath.Separator), -1)
	return t
}

func parseMatchList(list string) []string {
	// eg. "", "downloader,fetcher", "sync"
	if len(list) == 0 {
		return nil
	}
	ll := strings.Trim(list, " ")
	return strings.Split(ll, ",")
}

func getNonRecursivePackageName(s string) string {
	out := strings.TrimSuffix(s, string(filepath.Separator)+"...")
	out = strings.TrimSuffix(out, "...")
	return out
}

func handleLine(s string) (*test, error) {
	var t *test
	ss := strings.Trim(s, " ")
	if len(ss) == 0 {
		return nil, errEmptyLine
	}
	if strings.HasPrefix(ss, commentPattern) {
		return nil, errCommentLine
	}
	if strings.Contains(ss, commentPattern) {
		sss := strings.Split(ss, commentPattern)
		ss = strings.Trim(sss[0], " ")
	}
	t = parseLinePackageTest(ss)
	return t, nil
}

func lineMatchList(line string, whites, blacks []string) bool {
	if blacks != nil && len(blacks) > 0 {
		for _, m := range blacks {
			if strings.Contains(line, m) {
				return false
			}
		}
	}
	if whites != nil && len(whites) > 0 {
		for _, m := range whites {
			if !strings.Contains(line, m) {
				return false
			} else {
				return true
			}
		}
	}
	return true
}

func collectTestsFromFile(f string) (tests []*test, err error) {
	file, err := os.Open(f)
	if err != nil {
		return tests, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		t, e := handleLine(scanner.Text())
		if e == nil {
			tests = append(tests, t)
		}
	}

	return tests, scanner.Err()
}

func filterTests(tests []*test, allowed func(*test) bool) []*test {
	var out []*test
	for _, t := range tests {
		if allowed(t) {
			out = append(out, t)
		}
	}
	return out
}

func grepFailures(gotestout []byte) []string {
	reader := bytes.NewReader(gotestout)
	scanner := bufio.NewScanner(reader)

	var fails []string

	for scanner.Scan() {
		// eg. '--- FAIL: TestFastCriticalRestarts64 (12.34s)'
		text := scanner.Text()
		if !strings.Contains(text, "FAIL") {
			continue
		}
		if !strings.Contains(text, ":") {
			continue
		}
		step1 := strings.Split(text, ":")
		step2 := strings.Split(step1[1], "(")
		testname := strings.Trim(step2[0], " ")
		fails = append(fails, testname)
	}

	if e := scanner.Err(); e != nil {
		log.Fatal(e)
	}

	return fails
}

func runTest(t *test) ([]byte, error) {
	args := fmt.Sprintf("test %s", t.pkg)
	if t.name != "" {
		args += fmt.Sprintf(" -run %s", t.name)
	}
	log.Println("|", commandPrefix[0], commandPrefix[1], goExecutablePath+" "+args)
	cmd := exec.Command(commandPrefix[0], commandPrefix[1], goExecutablePath+" "+args)
	t.trials++
	out, err := cmd.CombinedOutput()
	return out, err
}

func tryIndividualTest(t *test, c chan error) {
	for t.trials < trialsAllowed {
		start := time.Now()
		if o, e := runTest(t); e == nil {
			log.Println(t)
			log.Printf("- PASS (%v) %d/%d", time.Since(start), t.trials, trialsAllowed)
			c <- nil
			return
		} else {
			log.Println(t)
			log.Printf("- FAIL (%v) %d/%d: %v", time.Since(start), t.trials, trialsAllowed, e)
			fmt.Println()
			fmt.Println(string(o))
		}
	}
	c <- fmt.Errorf("FAIL %s %s", t.pkg, t.name)
}

// only gets to send one nil/error on the given channel
func tryPackageTest(t *test, c chan error) {
	start := time.Now()
	if o, e := runTest(t); e == nil {
		log.Println(t)
		log.Printf("- PASS (%v)", time.Since(start))
		fmt.Println()
		fmt.Println(string(o))
		c <- nil
		return
	} else {
		log.Println(t)
		log.Printf("- FAIL (%v)", time.Since(start))
		fmt.Println()
		fmt.Println(string(o))

		fails := grepFailures(o)
		if len(fails) == 0 {
			log.Fatalf("%s reported failure, but no failing tests were discovered, err=%v",
				getNonRecursivePackageName(t.pkg), e)
		}

		var failingTests []*test
		for _, f := range fails {
			failingTests = append(failingTests,
				&test{
					pkg:    getNonRecursivePackageName(t.pkg),
					name:   f,
					trials: 1,
				})
		}
		log.Printf("Found failing test(s) in %s: %v. Rerunning...",
			getNonRecursivePackageName(t.pkg),
			fails,
		)

		pc := make(chan error, len(failingTests))
		for _, f := range failingTests {
			go tryIndividualTest(f, pc)
		}
		for i := 0; i < len(failingTests); i++ {
			if e := <-pc; e != nil {
				c <- e
				return
			}
		}
		c <- nil
	}
}

func tryTest(t *test, c chan error) {
	if t.name != "" {
		tryIndividualTest(t, c)
	} else {
		tryPackageTest(t, c)
	}
}

func Run(testsFile, whitelistMatch, blacklistMatch string, trialsN int) {
	e := run(testsFile, whitelistMatch, blacklistMatch, trialsN)
	if e != nil {
		log.Fatal(e)
	}
}

func run(testsFile, whitelistMatch, blacklistMatch string, trialsN int) error {
	if trialsN == 0 {
		return fmt.Errorf("trials allowed must be >0, got: %d", trialsAllowed)
	}
	trialsAllowed = trialsN

	whites := parseMatchList(whitelistMatch)
	blacks := parseMatchList(blacklistMatch)

	testsFile = filepath.Clean(testsFile)
	testsFile, _ = filepath.Abs(testsFile)

	allowed := func(t *test) bool {
		return lineMatchList(t.pkg+" "+t.name, whites, blacks)
	}

	alltests, err := collectTestsFromFile(testsFile)
	if err != nil {
		return err
	}

	tests := filterTests(alltests, allowed)

	log.Println("* go executable path:", goExecutablePath)
	log.Println("* command prefix:", strings.Join(commandPrefix, " "))
	log.Println("* tests file:", testsFile)
	log.Println("* trials allowed: ", trialsAllowed)
	log.Println("* blacklist: ", blacks)
	log.Println("* whitelist: ", whites)
	log.Printf("* running %d/%d tests", len(tests), len(alltests))

	var results = make(chan error, len(tests))

	allstart := time.Now()
	defer func() {
		log.Printf("FINISHED (%v)", time.Since(allstart))
	}()

	for _, t := range tests {
		go tryTest(t, results)
	}

	for i := 0; i < len(tests); i++ {
		if e := <-results; e != nil {
			return e
		}
	}

	close(results)
	return nil
}
