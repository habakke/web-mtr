CMD=web-mtr
BINARY=web-mtr
ROOT_DIR := $(if $(ROOT_DIR),$(ROOT_DIR),$(shell git rev-parse --show-toplevel))
BUILD_DIR = $(ROOT_DIR)/build
all: build

build: clean
	GOOS=linux GOARCH=386 go build -o $(BUILD_DIR)/$(BINARY) ./cmd/$(CMD)

docker: build
	docker build -t habakke/web-mtr:latest .
	docker push habakke/web-mtr

start:
	go run $(ROOT_DIR)/cmd/$(CMD)

clean:
	rm -rf $(BUILD_DIR)/$(BINARY)
