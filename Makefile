.DEFAULT_GOAL := build
APP_NAME = srs-resolver
BUILD_DIR = bin
INSTALL_DIR = /usr/local/bin
CONFIG_DIR = /etc/srs-resolver
LOG_DIR = /var/log

# -s — without symbol table
# -w — without DWARF debugging information
# This can reduce the file size by 50–70%!
LD_FLAGS= -s -w

fmt:
	goimports -l -w .
.PHONY:fmt

lint: fmt
	staticcheck ./...
.PHONY:lint

vet: fmt
	go vet ./...
.PHONY:vet

build: vet
	mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LD_FLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)

install:
	install -m 755 $(BUILD_DIR)/$(APP_NAME) $(INSTALL_DIR)/$(APP_NAME)
	mkdir -p $(CONFIG_DIR)
	mkdir -p $(LOG_DIR)
	install -m 644 config/srs-resolver.conf $(CONFIG_DIR)/
	install -m 664 /dev/null $(LOG_DIR)/srs-resolver.log

clean:
	rm -rf $(BUILD_DIR)

.PHONY:build install clean
