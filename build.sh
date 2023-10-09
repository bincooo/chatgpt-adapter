#!/bin/bash

cmd="go build"
args="-gcflags=-trimpath=$(pwd) -asmflags=-trimpath=$(pwd)"
outdir="bin"
rm -rf ${outdir}
GOOS=darwin GOARCH=amd64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/mac-exec cmd/exec.go
GOOS=darwin GOARCH=amd64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/mac-exec cmd/exec.go
GOOS=windows GOARCH=amd64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/win-exec.exe cmd/exec.go
GOOS=linux GOARCH=amd64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/linux-exec cmd/exec.go
GOARM=7 GOOS=linux GOARCH=arm64 ${cmd} ${args} -ldflags '-w -s' -o ${outdir}/linux-exec-arm64 cmd/exec.go

cp lang.toml ${outdir}/lang.toml
cp .env.example ${outdir}/.env.example