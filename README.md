# Schroedinger

Nondeterministic tests are unreliable and an enormous pain in the ass,
especially when it comes to continuous integration. There are a couple of
solutions:

1. __Fix your tests.__ If circumstances (erm... natural distaster, mental
   incapacity, tangly legacy code...) prevent you from doing so, continue to
   option 2. But you should still fix your tests.

2. __`schroedinger`__ Tell schroedinger how many times to try running your nondeterministic tests and
   he will diligently do so until the limit is reached or the test passes. He
   can run a whole package's tests and then single out individual failing tests, or
   run tests individually from the start. Original go test output (whether
   failing or successful) will be logged along with the command used so you can
   proofread the process.

## Install

```
go get -v github.com/etcdevteam/go-schroedinger/cmd/schroedinger/...
```

## Usage

1. You'll need a file listing tests for schroedinger to run. See
   [example.txt](./example.txt) for, well, an example. Note that this example file is used in schroedinger's own tests. Philosopher-approved.

2. Run schroedinger.

```
$ schroedinger -f example.txt
```

Command line options:

- `-f [STRING]` Can be either a relative or absolute path to the file with the
  tests listed in it. __Cannot be empty.__
- `-t [INTEGER]` Number of times to try a failing test before giving up.
  Default is 3.
- `-b [STRING]` Comma-separated __blacklist__ of patterns used to exclude tests _as written in the file_.
- `-w [STRING]` Comma-separated __whitelist__ of patterns to include as matched
  against the lines _in the tests file_.

