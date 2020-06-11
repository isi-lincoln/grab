GO = go
GOGET = $(GO) get -u

all: build/grab

build/grab: build
	go build -o $@ cmd/grab/main.go

build:
	mkdir build

.PHONY: clean
clean:
	rm -r -f build
