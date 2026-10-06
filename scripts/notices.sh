#!/bin/sh
# Generates THIRD-PARTY-NOTICES.md from the vendored dependencies.
#
# Every dependency is MIT, BSD, ISC or Apache-2.0. All four require the
# copyright notice and licence text to travel with any distribution of the
# binary, source or not — so this file has to ship inside the app bundle and
# the installer, not just live in the repository.
set -eu

cd "$(dirname "$0")/.."
out=THIRD-PARTY-NOTICES.md

{
    echo "# Third-party notices"
    echo
    echo "SD Card Sniffer bundles the following open source components."
    echo "Each is reproduced below with its original licence."
    echo
    echo "## Modifications"
    echo
    echo "\`github.com/diskfs/go-diskfs\` is shipped with local modifications to its"
    echo "ext4 reader, so that filesystems written without the \`metadata_csum\`"
    echo "feature can be read. The changes are recorded in \`patches/\` in the source"
    echo "distribution and are marked inline with \`PATCH(bbsdcardsniffer)\`."
    echo
    echo "## Components"
    echo

    find vendor \( -name "LICENSE*" -o -name "COPYING*" \) | sort | while read -r file; do
        module=$(echo "$file" | sed 's|^vendor/||; s|/LICENSE.*$||; s|/COPYING.*$||')
        echo "### $module"
        echo
        echo '```'
        cat "$file"
        echo '```'
        echo
    done
} > "$out"

echo "wrote $out ($(wc -l < "$out" | tr -d ' ') lines, $(grep -c '^### ' "$out") components)"
