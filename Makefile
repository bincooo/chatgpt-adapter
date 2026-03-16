EXEC := server
ENV := CGO_ENABLED=0

ifeq ($(OS),Windows_NT)
	ENV := SET ${ENV}
	PATCH_CMD := powershell -ExecutionPolicy Bypass -File patch/run.ps1
else
	PATCH_CMD := bash patch/run.sh
endif

.PHONY: all changelog clean patch

all: clean patch linux linux-arm64 macos windows

changelog:
	conventional-changelog -p angular -o CHANGELOG.md -w -r 0

clean:
	go clean -cache

patch:
	@$(PATCH_CMD) patch

linux:
	${ENV} GOARCH=amd64 GOOS=linux go build $(argv) -ldflags="-s -w" -o bin/linux/${EXEC} -trimpath main.go

linux-arm64:
	${ENV} GOARCH=arm64 GOOS=linux go build $(argv) -ldflags="-s -w" -o bin/linux/${EXEC}-arm64 -trimpath main.go

macos:
	${ENV} GOARCH=amd64 GOOS=darwin go build  $(argv) -ldflags="-s -w" -o bin/osx/${EXEC} -trimpath main.go

windows:
	${ENV} GOARCH=amd64 GOOS=windows & go build  -toolexec iocgo $(argv) -ldflags="-s -w" -o bin/windows/${EXEC}.exe -trimpath main.go