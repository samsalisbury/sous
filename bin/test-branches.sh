#!/bin/bash

LIST_FILE="$@"
if [ -z "$LIST_FILE" ]; then
	echo "usage: $0 <list-file> (file containing newline-separated branch names)"
	exit 1
fi

OUT_DIR_BASE="$(mktemp -d)"
mkdir -p "$OUT_DIR_BASE"
BRANCH_REV_FILE="$OUT_DIR_BASE/branch-list"
while read -r BRANCH; do
	if ! (git branch | grep "\\b$BRANCH\\b" "$@" > /dev/null); then
		echo "No local branch named $BRANCH"
		exit 1
	else
		REVISION="$(git rev-parse "$BRANCH")"
		echo "Found branch $BRANCH @$REVISION "
		echo "$BRANCH:$REVISION" >> "$BRANCH_REV_FILE"
	fi
done < "$@"

RUN_NUMBER=0
while ! [ $RUN_NUMBER = 100 ]; do
	RUN_NUMBER=$(( RUN_NUMBER + 1 ))
	while read -r BRANCH_REV; do
		BRANCH="$(echo "$BRANCH_REV" | cut -d':' -f1)"
		REVISION="$(echo "$BRANCH_REV" | cut -d':' -f2)"
		OUT_DIR="$OUT_DIR_BASE/test-branches/$BRANCH-$REVISION"
		OUT_DIR_PASS="$OUT_DIR/pass"
		OUT_DIR_FAIL="$OUT_DIR/fail"
		mkdir -p "$OUT_DIR_PASS"
		mkdir -p "$OUT_DIR_FAIL"
		git checkout "$BRANCH"
		echo
		echo
		echo
		echo "See results in $OUT_DIR_BASE"
		echo "Testing branches: $BRANCHES"
		echo "Starting run on $BRANCH ..."
		echo
		echo
		echo
		sleep 3
		OUT_FILE="integration-test-run-$RUN_NUMBER"
		OUT_PATH="$OUT_DIR/$OUT_FILE"
		PASS_PATH="$OUT_DIR_PASS/$OUT_FILE"
		FAIL_PATH="$OUT_DIR_FAIL/$OUT_FILE"
		if make test-integration 2>&1 | tee "$OUT_PATH"; then
			ln -s "$OUT_PATH" "$PASS_PATH"
		else
			ln -s "$OUT_PATH" "$FAIL_PATH"
		fi
	done < "$BRANCH_REV_FILE"
done
