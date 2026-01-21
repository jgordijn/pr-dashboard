.PHONY: build install clean run test

GO ?= go
BINARY := pr-dashboard
INSTALL_DIR := $(HOME)/.local/bin

build:
	$(GO) build -o $(BINARY) ./cmd/pr-dashboard

install: build
	mkdir -p $(INSTALL_DIR)
	install -m 0755 $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"

clean:
	rm -f $(BINARY)

run: build
	./$(BINARY)

test:
	$(GO) test ./...
