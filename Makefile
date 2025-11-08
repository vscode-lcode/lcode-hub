build:
	CGO_ENABLED=0 go build -ldflags="-X 'main.Version=$$(git describe --tags --always --dirty)' -s -w" -o lcode-hub .
deb: build
	nfpm pkg --packager deb
exe:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-X 'main.Version=$$(git describe --tags --always --dirty)' -s -w" -o nsi/lcode-hub.exe .
exe-installer: exe
	cd nsi && makensis installer.nsi
