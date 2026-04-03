BINARY_NAME := nuon-ext-webhook-server
BUILD_DIR := .

.PHONY: build clean

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

clean:
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
