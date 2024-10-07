#!/bin/bash

go mod edit -go "$(go version | { read -r _ _ v _; echo "${v#go}"; })"
go get -u ./
go mod tidy

file="ASnake"
go build -o "$file" . && echo -e "\033[32mBuild: $file\033[0m" || echo -e "\033[31mFailed: $file\033[0m"

file="ASnakeServer"
go build -o "$file" ./server && echo -e "\033[32mBuild: $file\033[0m" || echo -e "\033[31mFailed: $file\033[0m"
