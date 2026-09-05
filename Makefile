BINARY_NAME=ports
BIN_DIR=bin
VERSION?=0.1.0
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

.PHONY: all build test lint install clean

all: lint test build

build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/ports

test:
	go test -v -count=1 ./...

lint:
	go vet ./...

completion: build
	@mkdir -p $(HOME)/.config/fish/completions 2>/dev/null && \
	$(BIN_DIR)/$(BINARY_NAME) completion fish > $(HOME)/.config/fish/completions/$(BINARY_NAME).fish 2>/dev/null || true
	@mkdir -p $(HOME)/.zsh/completion 2>/dev/null && \
	$(BIN_DIR)/$(BINARY_NAME) completion zsh > $(HOME)/.zsh/completion/_$(BINARY_NAME) 2>/dev/null || true

install: build completion
	install -m 755 $(BIN_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME) 2>/dev/null || \
	install -m 755 $(BIN_DIR)/$(BINARY_NAME) $(HOME)/.local/bin/$(BINARY_NAME)

setcap: install
	sudo setcap cap_sys_ptrace+ep /usr/local/bin/$(BINARY_NAME) 2>/dev/null || \
	sudo setcap cap_sys_ptrace+ep $(HOME)/.local/bin/$(BINARY_NAME)

clean:
	rm -rf $(BIN_DIR)
