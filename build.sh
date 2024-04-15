#!/bin/bash

cmd="go build"
args="-gcflags=-trimpath=$(pwd) -asmflags=-trimpath=$(pwd)"
outdir="bin"
rm -rf ${outdir}
GOOS=darwin GOARCH=amd64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/mac-server server.go
GOOS=windows GOARCH=amd64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/win-server.exe server.go
GOOS=linux GOARCH=amd64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/linux-server server.go
GOARM=7 GOOS=linux GOARCH=arm64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/linux-server-arm64 server.go

# cp .env.example ${outdir}/.env.example
cp config.yaml $outdir/config.yaml
