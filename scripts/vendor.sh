#!/bin/sh
# Re-vendors dependencies and re-applies the local go-diskfs patches.
#
# go mod vendor overwrites vendor/ wholesale, so any direct edit there is lost.
# Run this instead of `go mod vendor` after changing dependencies.
set -eu

cd "$(dirname "$0")/.."

go mod vendor

for patch in patches/*.patch; do
    echo "applying $patch"
    git apply --directory=vendor/github.com/diskfs/go-diskfs "$patch" 2>/dev/null \
        || patch -p1 -d vendor/github.com/diskfs/go-diskfs < "$patch"
done

go build ./...
echo "vendor tree rebuilt and patched"
