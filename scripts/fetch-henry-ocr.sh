#!/bin/sh
# Download the Internet Archive's OCR (_djvu.xml) of the two scans that
# sources/henry-sermons points into, for TestHenryLeavesHoldTheirHeads.
# The files go to .archive-cache/ (git-ignored) or $SERMON_INDEX_ARCHIVE_CACHE;
# nothing of them is committed. 115 MB and 39 MB. A file already there is kept.
set -eu
here=$(cd "$(dirname "$0")/.." && pwd)
dir=${SERMON_INDEX_ARCHIVE_CACHE:-$here/.archive-cache}
mkdir -p "$dir"
for item in miscellaneouswor00henr completeworksofr0001revm; do
	f="$dir/${item}_djvu.xml"
	if [ -s "$f" ]; then
		echo "have $f"
		continue
	fi
	echo "fetching $item"
	curl -fsSL --retry 3 -o "$f.part" "https://archive.org/download/$item/${item}_djvu.xml"
	mv "$f.part" "$f"
done
