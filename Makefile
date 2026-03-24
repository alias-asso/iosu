iosud: cmd/iosud/main.go
	go build -o iosud cmd/iosud/main.go

iosu: cmd/iosu/main.go
	go build -o iosu cmd/iosu/main.go

clean:
	rm iosu iosud

all: iosud iosu
