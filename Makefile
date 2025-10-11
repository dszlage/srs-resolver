.DEFAULT_GOAL := build
APP_NAME = srs-resolver
BUILD_DIR = bin

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

install: build
	install -m 755 $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/$(APP_NAME)

clean:
	rm -rf $(BUILD_DIR)

.PHONY:build install clean
