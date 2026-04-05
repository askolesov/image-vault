MODULE ?= github.com/askolesov/image-vault
PACKAGE ?= ./...
OUTPUT ?= build/

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null)
BRANCH ?= $(shell git symbolic-ref -q --short HEAD 2>/dev/null)
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE ?= $(shell date +%FT%T%z)

GOARGS += -v
LDFLAGS += -s -w -X ${MODULE}/internal/buildinfo.version=${VERSION} \
	-X ${MODULE}/internal/buildinfo.commitHash=${COMMIT_HASH} \
	-X ${MODULE}/internal/buildinfo.buildDate=${BUILD_DATE} \
	-X ${MODULE}/internal/buildinfo.branch=${BRANCH}

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