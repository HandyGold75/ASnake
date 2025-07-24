FILE := ASnake.bin
TARGET := .

get:
	go get go@latest
	go get -u
	go mod tidy
	for indirect in $(tail +3 go.mod | grep "// indirect" | awk '{if ($1 =="require") print $2; else print $1;}'); do go get -u "${indirect}"; done
	go get -u
	go mod tidy

build:
	go build -o "$(TARGET)/$(FILE)" .

run:
	go build -o "$(TARGET)/$(FILE)" . && exec "$(TARGET)/$(FILE)"

clean:
	rm -f "$(TARGET)/$(FILE)"
	rm -f "$(TARGET)/ASnake.log"
