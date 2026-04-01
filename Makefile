BIN := notesview
BUILD_DIR := bin

.PHONY: build test lint clean

build:
	go build -o $(BUILD_DIR)/$(BIN) ./cmd/$(BIN)

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)
