build:
	CGO_ENABLED=0 go build -ldflags="-X 'main.Version=$$(git describe --tags --always --dirty)' -s -w" -o lcode-hub .
deb: build
	nfpm pkg --packager deb