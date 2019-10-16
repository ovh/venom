#!/bin/bash

cd cli/venom
for DIST in `go tool dist list | grep -v '^android/' | grep -v '^dragonfly/' | grep -v '^illumos/' | grep -v 'mips' | grep -v 'riscv64' | grep -v 'ppc64' | grep -v 's390x'  | grep -v '^nacl/' | grep -v '^plan9/' | grep -v '^darwin/arm' | grep -v '^js/wasm' | grep -v 'bsd/arm64' | grep -v 'windows/arm' `; do
    GOOS=`echo ${DIST} | cut -d / -f 1`
    GOARCH=`echo ${DIST} | cut -d / -f 2`

    architecture="${GOOS}-${GOARCH}"
    echo "Building ${architecture} ${path}"
    export GOOS=$GOOS
    export GOARCH=$GOARCH

    CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "-X github.com/ovh/venom.Version=${GIT_DESCRIBE}" -o bin/venom.${architecture}
done
