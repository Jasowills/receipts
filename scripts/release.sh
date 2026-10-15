#!/bin/sh
# Local release helper: format, vet, test, cross compile all four
# targets, and write SHA-256 checksums. Tagging and publishing stay
# manual until the owner, name, and license decisions land.
set -eu
test -z "$(gofmt -l .)"
go vet ./...
go test ./...
mkdir -p dist
rm -f dist/checksums.txt
for target in "darwin arm64" "darwin amd64" "linux amd64" "linux arm64"; do
  set -- $target
  out="dist/receipts-${1}-${2}"
  echo "building $out"
  GOOS="$1" GOARCH="$2" CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$out" ./cmd/receipts
done
(cd dist && shasum -a 256 receipts-* > checksums.txt)
cat dist/checksums.txt
