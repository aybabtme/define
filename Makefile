
all:
	go build

install:
	go install

run:
	go run define.go

clean:
	rm -rf db
	rm define
