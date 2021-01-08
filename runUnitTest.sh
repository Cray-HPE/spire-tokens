#!/bin/bash

# Copyright 2020 Hewlett Packard Enterprise Development LP
# NOTE -- The following syntax will run unit tests without worrying
# about code coverage. To see an example of code coverage (which re-runs
# these same unit tests) see runCoverage.sh. The same tests are getting
# run in both places.

echo "Unit Tests for Go"
TEST_OUTPUT_DIR="$PWD/build/results/unittest"
mkdir -p $TEST_OUTPUT_DIR
export GOPATH="$HOME/go"
export PATH="$PATH:$GOPATH/bin"
go get -u github.com/jstemmer/go-junit-report
go test ./tests -v | go-junit-report > "$TEST_OUTPUT_DIR/testing.xml"
