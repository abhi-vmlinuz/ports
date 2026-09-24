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

PREFIX?=/usr
BINDIR?=$(PREFIX)/bin
SHAREDIR?=$(PREFIX)/share

install: build
	@if [ "$$(id -u)" -eq 0 ]; then \
		install -d $(BINDIR); \
		install -m 755 $(BIN_DIR)/$(BINARY_NAME) $(BINDIR)/$(BINARY_NAME); \
		mkdir -p $(SHAREDIR)/bash-completion/completions 2>/dev/null && $(BIN_DIR)/$(BINARY_NAME) completion bash > $(SHAREDIR)/bash-completion/completions/$(BINARY_NAME) 2>/dev/null || true; \
		mkdir -p $(SHAREDIR)/zsh/site-functions 2>/dev/null && $(BIN_DIR)/$(BINARY_NAME) completion zsh > $(SHAREDIR)/zsh/site-functions/_$(BINARY_NAME) 2>/dev/null || true; \
		mkdir -p $(SHAREDIR)/fish/vendor_completions.d 2>/dev/null && $(BIN_DIR)/$(BINARY_NAME) completion fish > $(SHAREDIR)/fish/vendor_completions.d/$(BINARY_NAME).fish 2>/dev/null || true; \
	else \
		sudo install -d $(BINDIR); \
		sudo install -m 755 $(BIN_DIR)/$(BINARY_NAME) $(BINDIR)/$(BINARY_NAME); \
		sudo mkdir -p $(SHAREDIR)/bash-completion/completions 2>/dev/null && $(BIN_DIR)/$(BINARY_NAME) completion bash | sudo tee $(SHAREDIR)/bash-completion/completions/$(BINARY_NAME) >/dev/null 2>&1 || true; \
		sudo mkdir -p $(SHAREDIR)/zsh/site-functions 2>/dev/null && $(BIN_DIR)/$(BINARY_NAME) completion zsh | sudo tee $(SHAREDIR)/zsh/site-functions/_$(BINARY_NAME) >/dev/null 2>&1 || true; \
		sudo mkdir -p $(SHAREDIR)/fish/vendor_completions.d 2>/dev/null && $(BIN_DIR)/$(BINARY_NAME) completion fish | sudo tee $(SHAREDIR)/fish/vendor_completions.d/$(BINARY_NAME).fish >/dev/null 2>&1 || true; \
	fi
	@if [ -n "$$SUDO_USER" ]; then \
		UHOME=$$(getent passwd "$$SUDO_USER" 2>/dev/null | cut -d: -f6); \
		if [ -n "$$UHOME" ] && [ -d "$$UHOME" ]; then \
			mkdir -p "$$UHOME/.config/fish/completions" 2>/dev/null && $(BIN_DIR)/$(BINARY_NAME) completion fish > "$$UHOME/.config/fish/completions/$(BINARY_NAME).fish" 2>/dev/null || true; \
			mkdir -p "$$UHOME/.zsh/completion" 2>/dev/null && $(BIN_DIR)/$(BINARY_NAME) completion zsh > "$$UHOME/.zsh/completion/_$(BINARY_NAME)" 2>/dev/null || true; \
			chown -R "$$SUDO_USER:" "$$UHOME/.config/fish/completions/$(BINARY_NAME).fish" "$$UHOME/.zsh/completion/_$(BINARY_NAME)" 2>/dev/null || true; \
		fi \
	fi

setcap: install
	@if [ "$$(id -u)" -eq 0 ]; then \
		setcap cap_sys_ptrace+ep $(BINDIR)/$(BINARY_NAME); \
	else \
		sudo setcap cap_sys_ptrace+ep $(BINDIR)/$(BINARY_NAME); \
	fi

clean:
	rm -rf $(BIN_DIR)
