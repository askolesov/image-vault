MODULE ?= github.com/askolesov/image-vault
PACKAGE ?= ./...
OUTPUT ?= build/

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null)
BRANCH ?= $(shell git symbolic-ref -q --short HEAD 2>/dev/null)
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE ?= $(shell date +%FT%T%z)

GOARGS += -v
LDFLAGS += -s -w -X ${MODULE}/pkg/buildinfo.version=${VERSION} \
	-X ${MODULE}/pkg/buildinfo.commitHash=${COMMIT_HASH} \
	-X ${MODULE}/pkg/buildinfo.buildDate=${BUILD_DATE} \
	-X ${MODULE}/pkg/buildinfo.branch=${BRANCH}

.PHONY: run
run:
	go run ${GOARGS} -tags "${GOTAGS}" -ldflags "${LDFLAGS}" ${PACKAGE}

.PHONY: build
build:
	go build ${GOARGS} -tags "${GOTAGS}" -ldflags "${LDFLAGS}" -o ${OUTPUT} ${PACKAGE}

.PHONY: install
install:
	go install ${GOARGS} -tags "${GOTAGS}" -ldflags "${LDFLAGS}" ${PACKAGE}

.PHONY: test
test:
	go test -count=1 -v ./...

.PHONY: build-clean
build-clean:
	rm -rf ${OUTPUT}

.PHONY: lint
lint:
	golangci-lint run -v

.PHONY: pre-push
pre-push:
	go mod tidy
	make lint
	make test
	make build

.PHONY: test-changes
test-changes:
	git diff --name-only --cached --diff-filter=ACM | grep -E '\.go$$' | xargs -n 1 gofmt -w
	git diff --cached --name-only --diff-filter=ACM | grep -E '\.go$$' | xargs -n 1 go vet
	git diff --cached --name-only --diff-filter=ACM | grep -E '\.go$$' | xargs -n 1 go test -v
