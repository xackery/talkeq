VERSION ?= v1.3.13
NAME := talkeq

# run a copy of talkeq
run: sanitize
	@echo "run: building"
	mkdir -p bin
	cd bin && go run ../main.go

# clean up and check for errors
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

# do tests against the codebase
test:
	@go test -cover ./...

# build all supported versions
build-all: build-prepare build-linux build-darwin build-windows

# prep for building
build-prepare:
	@echo "Preparing talkeq ${VERSION}"
	@rm -rf bin/*
	@-mkdir -p bin/


# make a darwin binary
build-darwin:
	@echo "build-darwin: building ${VERSION}"
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -buildmode=pie -ldflags="-X main.Version=${VERSION} -s -w" -o bin/${NAME}-darwin main.go

# make a linux binary
build-linux:
	@echo "build-linux: building ${VERSION}"
	go env
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X main.Version=${VERSION} -s -w" -o bin/${NAME}-linux main.go

#make a windows binary
build-windows:
	@echo "build-windows: building ${VERSION}"
	GOOS=windows GOARCH=amd64 go build -buildmode=pie -ldflags="-X main.Version=${VERSION} -s -w" -o bin/${NAME}-windows.exe main.go
	@#GOOS=windows GOARCH=386 go build -buildmode=pie -ldflags="-X main.Version=${VERSION} -s -w" -o bin/${NAME}-windows-x86.exe main.go

# analyze the binary using binskim
analyze:
	binskim analyze bin/${NAME}-linux

# CICD triggers this
set-version-%:
	@echo "VERSION=${VERSION}.$*" >> $$GITHUB_ENV

# run pprof and dump 3 snapshots of heap
profile-heap:
	@echo "profile-heap: running pprof watcher for 2 minutes with snapshots 0 to 3..."
	@-mkdir -p bin
	curl http://localhost:8082/debug/pprof/heap > bin/heap.0.pprof
	sleep 30
	curl http://localhost:8082/debug/pprof/heap > bin/heap.1.pprof
	sleep 30
	curl http://localhost:8082/debug/pprof/heap > bin/heap.2.pprof
	sleep 30
	curl http://localhost:8082/debug/pprof/heap > bin/heap.3.pprof

# peek at a heap
profile-heap-%:
	@echo "profile-heap-$*: use top20, svg, or list *word* for pprof commands, ctrl+c when done"
	go tool pprof bin/heap.$*.pprof

# run a trace on quail
profile-trace:
	@echo "profile-trace: getting trace data, this can show memory leaks and other issues..."
	curl http://localhost:8082/debug/pprof/trace > bin/trace.out
	go tool trace bin/trace.out
