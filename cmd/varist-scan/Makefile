GIT_REV ?= $(shell git describe --tag --always --long --dirty)
LDFLAGS += -w -s -extldflags=-static
LDFLAGS := -X main.version="${GIT_REV}"
export CGO_ENABLED=0

.PHONY: go-varist-scan
go-varist-scan:
	go build -ldflags="$(LDFLAGS)" $(BUILD_ARGS) -o $@ .
