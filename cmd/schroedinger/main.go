package main

import (
	"flag"
	"log"
	"github.com/etcdevteam/go-schroedinger"
)

// tests should be specified with lines of the form:
// ./eth/downloader TestCanonicalSynchronisation
// or
// github.com/ethereumproject/go-ethereum/eth/downloader TestFastCriticalRestarts
// comments are allowed with the '#' character and usual usage
var testsFile string

// allowed times to try to get a nondeterministic test to pass
var trialsAllowed int

// string to match to *list tests
var whitelistMatch string
var blacklistMatch string

func init() {
	flag.StringVar(&testsFile, "f", "", "path file to file containing tests to run")
	flag.StringVar(&whitelistMatch, "w", "", "whitelist lines containing")
	flag.StringVar(&blacklistMatch, "b", "", "blacklist lines containing")
	flag.IntVar(&trialsAllowed, "t", 3, "allowed trials before nondeterministic test actually fails")
	flag.Parse()
}

func main() {
	if testsFile == "" {
		log.Fatal("testsfile cannot be empty")
	}
	if trialsAllowed < 1 {
		log.Fatal("trials allowed cannot be less than 1")
	}
	if (whitelistMatch != "" && blacklistMatch != "") && whitelistMatch == blacklistMatch {
		log.Fatal("whitelist cannot match blacklist")
	}
	schroedinger.Run(testsFile, whitelistMatch, blacklistMatch, trialsAllowed)
}