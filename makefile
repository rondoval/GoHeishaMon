GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=GoHeishaMon
BINARY_UNIX=$(BINARY_NAME)_AMD64
BINARY_MIPS=$(BINARY_NAME)_MIPS
BINARY_ARM=$(BINARY_NAME)_ARM
BINARY_MIPSUPX=$(BINARY_NAME)_MIPSUPX


all: build build-rpi build-linux build-mips upx

build: 
	$(GOBUILD) -o $(BINARY_NAME) -v
test: 
	$(GOTEST) -v ./...
clean: 
	$(GOCLEAN)
	rm -f dist/*
run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o dist/$(BINARY_UNIX)
build-mips:
	CGO_ENABLED=0 GOOS=linux GOARCH=mips GOMIPS=softfloat $(GOBUILD) -ldflags "-s -w" -a -o dist/$(BINARY_MIPS)
build-rpi:
	GOOS=linux GOARCH=arm GOARM=5 $(GOBUILD) -o dist/$(BINARY_ARM)
upx:
	upx -f --brute -o dist/$(BINARY_MIPSUPX) dist/$(BINARY_MIPS)
compilesquash:
	cp dist/$(BINARY_MIPSUPX) OS/RootFS/usr/bin/$(BINARY_MIPSUPX)
	cp config.yaml.example OS/RootFS/etc/gh/config.yaml
	cp topics.yaml OS/RootFS/etc/gh/topics.yaml

