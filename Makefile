TARGET_EXEC := server
ENV := CGO_ENABLED=0

AT := ""

ifeq ($(OS),Windows_NT)
	ENV := SET ${ENV}
	AT := &
endif

.PHONY: all changelog clean install build

all: clean install build-linux build-linux-arm64 build-osx build-win

changelog:
	conventional-changelog -p angular -o CHANGELOG.md -w -r 0

clean:
	go clean -cache

install: clean
	go install -ldflags="-s -w" -trimpath ./cmd/iocgo

build-linux:
	${ENV} GOARCH=amd64 GOOS=linux ${AT} go build -toolexec iocgo $(argv) -ldflags="-s -w" -o bin/linux/${TARGET_EXEC} -trimpath main.go

build-linux-arm64:
	${ENV} GOARCH=arm64 GOOS=linux ${AT} go build  -toolexec iocgo $(argv) -ldflags="-s -w" -o bin/linux/${TARGET_EXEC}-arm64 -trimpath main.go

build-osx:
	${ENV} GOARCH=amd64 GOOS=darwin ${AT} go build  -toolexec iocgo $(argv) -ldflags="-s -w" -o bin/osx/${TARGET_EXEC} -trimpath main.go

build-win:
	${ENV} GOARCH=amd64 GOOS=windows ${AT} go build  -toolexec iocgo $(argv) -ldflags="-s -w" -o bin/windows/${TARGET_EXEC}.exe -trimpath main.go