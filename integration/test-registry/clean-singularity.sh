#!/usr/bin/env bash

if [ $# -lt 1 ]; then
  echo "Usage: $0 <singularity host>"
  exit 1
fi

sing="$1"

for req in $(cygnus -H "$sing" | awk '{ gsub(/>/, "%3E"); print $1 }'); do
  echo "Cleaning $req"
  curl -X DELETE "$sing/api/requests/request/$req"
  echo
done

while [ "$(cygnus -H "$sing" | wc -l)" -gt 0 ]; do
  sleep 0.2
done
