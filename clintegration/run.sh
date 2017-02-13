#!/usr/bin/env bash

set -x

cd "$(dirname "$0")/.." || exit

rm -f doc/shellexamples/*
go test -v -timeout 15m ./clintegration/;
exec less "$(ls doc/shellexamples/*.blended | tail -n 1)"
