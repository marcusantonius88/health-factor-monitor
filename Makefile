GO_BIN_DIR ?= $(shell go env GOPATH)/bin

.PHONY: install uninstall

install:
	mkdir -p "$(GO_BIN_DIR)"
	go build -o "$(GO_BIN_DIR)/hf" ./cmd/hfmon

uninstall:
	rm -f "$(GO_BIN_DIR)/hf"