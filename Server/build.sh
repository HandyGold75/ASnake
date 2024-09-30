#!/bin/bash

cd Server ||

go mod edit -go "$(go version | { read -r _ _ v _; echo "${v#go}"; })"
go mod tidy
go get -u ./

file="ASnakeServer"

env GOOS=linux GOARCH=amd64 go build -o "$file.bin" . && echo -e "\033[32mBuild: $file.bin\033[0m" || echo -e "\033[31mFailed: $file.bin\033[0m"
