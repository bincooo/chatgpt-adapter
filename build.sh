#!/bin/bash

cmd="go build"
args="-gcflags=-trimpath=$(pwd) -asmflags=-trimpath=$(pwd)"
outdir="bin"
rm -rf ${outdir}
GOOS=darwin GOARCH=amd64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/mac-server cmd/exec.go
GOOS=windows GOARCH=amd64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/win-server.exe cmd/exec.go
GOOS=linux GOARCH=amd64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/linux-server cmd/exec.go
GOARM=7 GOOS=linux GOARCH=arm64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/linux-server-arm64 cmd/exec.go

cp .env.example ${outdir}/.env.example