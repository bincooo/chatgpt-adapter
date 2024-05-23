#!/bin/bash

outdir="bin"
rm -rf ${outdir}
GOOS=darwin GOARCH=amd64 go build -ldflags '-w -s' -o ${outdir}/mac-server -trimpath cmd/command.go
GOOS=windows GOARCH=amd64 go build -ldflags '-w -s' -o ${outdir}/win-server.exe -trimpath cmd/command.go
GOOS=linux GOARCH=amd64 go build -ldflags '-w -s' -o ${outdir}/linux-server -trimpath cmd/command.go
GOARM=7 GOOS=linux GOARCH=arm64 go build -ldflags '-w -s' -o ${outdir}/linux-server-arm64 -trimpath cmd/command.go
cp config.yaml $outdir/config.yaml
