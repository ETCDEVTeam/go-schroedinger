package schroedinger

import (
	"testing"
	"reflect"
	"math/rand"
	"time"
	"os"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestGrepFailures(t *testing.T) {
	var outputWithFails = `
ok  	github.com/ethereumproject/go-ethereum/p2p	0.395s
ok  	github.com/ethereumproject/go-ethereum/p2p/discover	6.374s
ok  	github.com/ethereumproject/go-ethereum/p2p/distip	0.014s
--- FAIL: TestUPNP_DDWRT (2.10s)
	natupnp_test.go:201: HTTPU request M-SEARCH *
	natupnp_test.go:201: HTTPU request M-SEARCH *
	natupnp_test.go:201: HTTPU request M-SEARCH *
	natupnp_test.go:201: HTTPU request M-SEARCH *
	natupnp_test.go:201: HTTPU request M-SEARCH *
	natupnp_test.go:201: HTTPU request M-SEARCH *
	natupnp_test.go:167: not discovered
FAIL
FAIL	github.com/ethereumproject/go-ethereum/p2p/nat	2.664s
`
	var outputOK = `
ok  	github.com/ethereumproject/go-ethereum/p2p	0.395s
ok  	github.com/ethereumproject/go-ethereum/p2p/discover	6.374s
ok  	github.com/ethereumproject/go-ethereum/p2p/distip	0.014s
`

	if failures := grepFailures([]byte(outputWithFails)); len(failures) != 1 {
		t.Errorf("got %v, want: %v", len(failures), 1)
	}
	if failures := grepFailures([]byte(outputOK)); len(failures) != 0 {
		t.Errorf("got %v, want: %v", len(failures), 0)
	}
}

func TestParseMatchList(t *testing.T) {
	cases := []struct{
		arg string
		want []string
	}{
		{arg: "", want: nil},
		{arg: "downloader,fetcher ", want: []string{"downloader", "fetcher"}},
	}
	for _, c := range cases {
		if got := parseMatchList(c.arg); (got == nil && c.want != nil) || len(got) != len(c.want) {
			t.Errorf("got: %v, want: %v", got, c.want)
		}
	}
}

func TestCat(t *testing.T) {
	if os.Getenv("thisIsOnlyATest") == "" {
		t.Skip("No peeking!")
	}
	aliveOrDead := rand.Float64()
	if aliveOrDead > 0.3 { // i mean, the odds could be worse
		t.Fatalf("Kitty? %.2f", aliveOrDead)
	}
}

func TestIntegration(t *testing.T) {
	//github.com/etcdevteam/go-schroedinger TestTest1
	//github.com/etcdevteam/go-schroedinger/...
	//github.com/etcdevteam/go-schroedinger

	want := []*test{
		{pkg: "github.com/etcdevteam/go-schroedinger", name: "TestCat"},
		{pkg: "github.com/etcdevteam/go-schroedinger/..."},
		{pkg: "github.com/etcdevteam/go-schroedinger"},
	}

	got, err := collectTests("./example.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got: %v, want: %v", got, want)
	}

	allowed := func(t *test) bool {
		if t.name == "" {
			return false
		}
		return true
	}

	filteredWant := want[:1]
	filteredGot := filterTests(got, allowed)
	if !reflect.DeepEqual(filteredGot, filteredWant) {
		t.Errorf("got: %v, want: %v", got, want)
	}

	os.Setenv("thisIsOnlyATest", "WTF")
	if e := run("./example.txt", "Cat", "", 20); e != nil {
		t.Fatal(e)
	}
	os.Setenv("thisIsOnlyATest", "")
}