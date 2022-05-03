#!/bin/bash

#
# MIT License
#
# (C) Copyright 2021-2022 Hewlett Packard Enterprise Development LP
#
# Permission is hereby granted, free of charge, to any person obtaining a
# copy of this software and associated documentation files (the "Software"),
# to deal in the Software without restriction, including without limitation
# the rights to use, copy, modify, merge, publish, distribute, sublicense,
# and/or sell copies of the Software, and to permit persons to whom the
# Software is furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included
# in all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
# THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
# OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
# ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
# OTHER DEALINGS IN THE SOFTWARE.
#

set -ex

docker run --rm -v "${WORKSPACE}":/mnt/workspace -w /mnt/workspace artifactory.algol60.net/docker.io/library/golang:alpine /bin/sh -c '
set -ex -o pipefail

echo "Unit Tests for Go"
TEST_OUTPUT_DIR="$PWD/build/results/unittest"
mkdir -p $TEST_OUTPUT_DIR
export GOPATH="$HOME/go"
export PATH="$PATH:$GOPATH/bin"
go get -u github.com/jstemmer/go-junit-report
apk update; apk add gcc musl-dev gcompat
go test -v | go-junit-report > "$TEST_OUTPUT_DIR/testing.xml"
'
