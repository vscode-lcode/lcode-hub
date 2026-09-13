VERSION := $(shell git describe --tags --always --dirty)

build:
	CGO_ENABLED=0 go build -ldflags="-X 'main.Version=${VERSION}' -s -w" -o lcode-hub .
deb: build
	VERSION=${VERSION} nfpm pkg --packager deb
exe:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-X 'main.Version=${VERSION}' -s -w" -o nsi/lcode-hub.exe .
exe-installer: exe
	echo '!define VERSION "$(VERSION)"' > nsi/version.nsh
	cd nsi && makensis installer.nsi
