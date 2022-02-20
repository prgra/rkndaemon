BINARY     := rkndaemon
BUILDFLAGS := "-s -w"
.PHONY: build

build:
	@mkdir -p build
	go build -ldflags $(BUILDFLAGS) -o build/$(BINARY)
	GOOS=linux GOARCH=amd64 go build -ldflags $(BUILDFLAGS) -o build/$(BINARY)-linux_amd64
	GOOS=linux GOARCH=arm64 go build -ldflags $(BUILDFLAGS) -o build/$(BINARY)-linux_arm64
	GOOS=freebsd GOARCH=amd64 go build -ldflags $(BUILDFLAGS) -o build/$(BINARY)-freebsd_amd64
	GOOS=freebsd GOARCH=arm64 go build -ldflags $(BUILDFLAGS) -o build/$(BINARY)-freebsd_arm64
