.PHONY = all clean

all: bitratemon bitratemon-linux-386 bitratemon-linux-amd64

clean:
	go clean
	rm bitratemon bitratemon-linux-386 bitratemon-linux-amd64

bitratemon: main.go
	go build

bitratemon-linux-386: main.go
	GOOS=linux GOARCH=386 go build -o $@

bitratemon-linux-amd64: main.go
	GOOS=linux GOARCH=amd64 go build -o $@
