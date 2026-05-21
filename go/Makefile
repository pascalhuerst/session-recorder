VERSION := $(shell git describe --all --long)
GO ?= go
GOBUILDFLAGS ?= -v

BINARIES = \
  session_recorder_server \

PACKAGES = \
  mdns

all: $(BINARIES)

session_recorder_server:
	${GO} build -o bin/$@ $(GOBUILDFLAGS) -ldflags "-X main.version=$(VERSION)" ./cmd/$@

clean:
	cd bin && rm -f ${BINARIES}

test:
	${GO} test -v -race ./...

test-coverage:
	${GO} test -v -race -coverprofile=coverage.out ./...
	${GO} tool cover -html=coverage.out -o coverage.html
