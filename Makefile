.PHONY: all build run gotool install clean help

BINARY_NAME=back_end_go
BIN_DIR=./bin/

all: gotool build

build:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ${BINARY_NAME}

run:
	@go run ./

gotool:
	go fmt ./
	go vet ./

install:
	mkdir -p ${BIN_DIR}
	make build
	mv ${BINARY_NAME} ${BIN_DIR}

clean:
	@if [ -f ${BINARY_NAME} ] ; then rm ${BINARY_NAME} ; fi
