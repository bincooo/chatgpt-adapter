#!/bin/bash
rm -rf ./*exec*
GOOS=darwin GOARCH=amd64 go build -o mac-exec cmd/exec.go
GOOS=windows GOARCH=amd64 go build -o win-exec.exe cmd/exec.go
GOOS=linux GOARCH=amd64 go build -o linux-exec cmd/exec.go
GOARM=7 GOOS=linux GOARCH=arm64 go build -o linux-exec-arm64 cmd/exec.go