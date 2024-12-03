TARGET_EXEC := server

.PHONY: all changelog clean install build

all: clean install build

changelog:
	conventional-changelog -p angular -o CHANGELOG.md -w -r 0

clean:
	go clean -cache

install: clean
	go install -ldflags="-s -w" -trimpath ./cmd/iocgo

build:
	go build -toolexec iocgo -ldflags="-s -w" -trimpath -o server .

build-linux:
	GOARCH=amd64 GOOS=linux go build -toolexec iocgo -ldflags="-s -w" -o bin/linux/${TARGET_EXEC} -trimpath main.go

build-linux-arm64:
	GOARCH=arm64 GOOS=linux go build  -toolexec iocgo -ldflags="-s -w" -o bin/linux/${TARGET_EXEC}-arm64 -trimpath main.go

build-osx:
	GOARCH=amd64 GOOS=darwin go build  -toolexec iocgo -ldflags="-s -w" -o bin/osx/${TARGET_EXEC} -trimpath main.go

build-windows:
	GOARCH=amd64 GOOS=windows go build  -toolexec iocgo -ldflags="-s -w" -o bin/windows/${TARGET_EXEC}.exe -trimpath main.go