#!/usr/bin/env bash

set -euo pipefail

# This script is not intended to be bomb-proof. It is designed to optimistically
# attempt to install the latest released version of Sous. If any of its assumptions
# about how GitHub works or the structure of the tarball become false, it will fail.

REPO_URL="https://github.com/opentable/sous"
PREFIX=/usr/local/sous
BIN=/usr/local/bin

[ -d "$PREFIX" ] || mkdir -p "$PREFIX"

if [ "$(uname)" == Darwin ]; then
	PLATFORM=darwin-amd64
elif [ "$(uname)" == Linux ]; then
	PLATFORM=linux-amd64
else
	echo "Unix name $(uname) not recognised."
	echo "Sous can be installed on Linux or Darwin (macOS)"
fi

# Get the git tag of the latest release by examining the HTTP redirect.
TAG=$(curl -si $REPO_URL/releases/latest | grep '^Location: ' | cut -f 8 -d /)
TAG="${TAG%?}"

if [ "${TAG:0:1}" != "v" ]; then
	echo "Expected tag to begin with 'v'."
fi

# VERSION is TAG without first character.
VERSION="${TAG#?}"

echo "Installing sous version $VERSION"

# Calculate assumed filename and tarball URL.
FILENAME="sous-${PLATFORM}_$VERSION"
TARBALL="$FILENAME.tar.gz"
TARBALL_URL="$REPO_URL/releases/download/$TAG/$TARBALL"

echo "Downloading sous from $TARBALL_URL"
curl -LO "$TARBALL_URL"
mv "$TARBALL" "$PREFIX/"

(
	cd "$PREFIX"
	echo "Extracting to $PREFIX..."
	tar -C "$PREFIX" -xvf "$TARBALL"
)

BIN_DIR="$PREFIX/$FILENAME"

[ -d "$BIN_DIR" ] # Fail if the expected dir doesn't exist.

echo "Adding symlink in $BIN/sous..."
BINARY="$BIN_DIR/sous"
ln -fs "$BINARY" "$BIN/sous"

echo "Sous $VERSION installed to $(which sous)"
