#!/bin/sh
# Regenerates the fixture PDFs from src/. Needs mutool.
set -e
cd "$(dirname "$0")"
mutool create -o basic.pdf src/basic-1.txt src/basic-2.txt
mutool create -o twocol.pdf src/twocol.txt
mutool create -o table.pdf src/table.txt
mutool create -o code.pdf src/code.txt
mutool create -o link.pdf src/link.txt
mutool run src/addlink.js link.pdf link.pdf 160 58 245 76 https://mupdf.com
mutool draw -q -r 150 -o scan.png basic.pdf 1
mutool create -o scan.pdf src/scan.txt
rm scan.png
