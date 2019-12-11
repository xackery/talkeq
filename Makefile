# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
VERSION := 0.0.1
NAME := talkeq

.PHONY: build-all
build-all:
	@echo "Preparing talkeq v${VERSION}"
	@rm -rf bin/*
	@-mkdir -p bin/
	@echo "Building Linux"
	@GOOS=linux GOARCH=amd64 go build -o bin/${NAME}-${VERSION}-linux-x64 main.go
	@GOOS=linux GOARCH=386 go build -o bin/${NAME}-${VERSION}-linux-x86 main.go
	@echo "Building Windows"
	@GOOS=windows GOARCH=amd64 go build -o bin/${NAME}-${VERSION}-win-x64.exe main.go
	@GOOS=windows GOARCH=386 go build -o bin/${NAME}-${VERSION}-win-x86.exe main.go
	@echo "Building OSX"
	@GOOS=darwin GOARCH=amd64 go build -o bin/${NAME}-${VERSION}-osx-x64 main.go
