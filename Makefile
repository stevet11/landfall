all: craft craft.exe

craft:
	go build -o craft .

craft.exe:
	GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui -s -w" -o craft.exe .

test:
	go run . -selftest

.PHONY: all craft craft.exe test
