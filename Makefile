all: landfall landfall.exe

landfall:
	go build -o landfall .

landfall.exe:
	GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui -s -w" -o landfall.exe .

test:
	go run . -selftest

.PHONY: all landfall landfall.exe test
