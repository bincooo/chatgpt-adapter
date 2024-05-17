#!/bin/bash

cmd="go build"
args="-gcflags=-trimpath=$(pwd) -asmflags=-trimpath=$(pwd)"
outdir="bin"
rm -rf ${outdir}
GOOS=darwin GOARCH=amd64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/mac-server cmd/command.go
GOOS=windows GOARCH=amd64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/win-server.exe cmd/command.go
GOOS=linux GOARCH=amd64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/linux-server cmd/command.go
GOARM=7 GOOS=linux GOARCH=arm64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/linux-server-arm64 cmd/command.go

# cp .env.example ${outdir}/.env.example
cp config.yaml $outdir/config.yaml
