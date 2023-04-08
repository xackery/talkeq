VERSION := v1.3.7
NAME := talkeq

# CICD triggers this
.PHONY: set-variable
set-version:
	@echo "VERSION=${VERSION}" >> $$GITHUB_ENV

sanitize:
	@echo "sanitize: checking for errors"
	rm -rf vendor/
	go vet -tags ci ./...
	test -z $(goimports -e -d . | tee /dev/stderr)
	gocyclo -over 30 .
	golint -set_exit_status $(go list -tags ci ./...)
	staticcheck -go 1.14 ./...
	go test -tags ci -covermode=atomic -coverprofile=coverage.out ./...
    coverage=`go tool cover -func coverage.out | grep total | tr -s '\t' | cut -f 3 | grep -o '[^%]*'`

run: sanitize
	@echo "run: building"
	mkdir -p bin
	cd bin && go run ../main.go

test:
	@go test -cover ./...
.PHONY: build-all
build-all: sanitize build-prepare build-linux build-darwin build-windows
.PHONY: build-prepare
build-prepare:
	@echo "Preparing talkeq ${VERSION}"
	@rm -rf bin/*
	@-mkdir -p bin/
.PHONY: build-darwin
build-darwin:
	@echo "build-darwin: building ${VERSION}"
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -buildmode=pie -ldflags="-X main.Version=${VERSION} -s -w" -o bin/${NAME}-${VERSION}-darwin-x64 main.go
.PHONY: build-linux
build-linux:
	@echo "build-linux: building ${VERSION}"
	go env
	GOOS=linux GOARCH=amd64 go build -buildmode=pie -ldflags="-X main.Version=${VERSION} -w -extldflags '-static'" -o bin/${NAME}-${VERSION}-linux-x64 main.go
.PHONY: build-windows
build-windows:
	@echo "build-windows: building ${VERSION}"
	GOOS=windows GOARCH=amd64 go build -buildmode=pie -ldflags="-X main.Version=${VERSION} -s -w" -o bin/${NAME}-${VERSION}-win-x64.exe main.go
	GOOS=windows GOARCH=386 go build -buildmode=pie -ldflags="-X main.Version=${VERSION} -s -w" -o bin/${NAME}-${VERSION}-win-x86.exe main.go
analyze:
	binskim analyze bin/${NAME}-${VERSION}-linux-x64