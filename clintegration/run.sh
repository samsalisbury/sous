#!/usr/bin/env bash

set -x

cd "$(dirname "$0")/.." || exit

integration/test-registry/clean-singularity.sh http://192.168.99.100:7099/singularity
rm -f doc/shellexamples/*
rm -rf "$TMPDIR/shell*";
go test -v -timeout 500s ./clintegration/;
exec less "$(ls doc/shellexamples/*.blended | tail -n 1)"
