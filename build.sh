#!/bin/bash

go mod edit -go "$(go version | { read -r _ _ v _; echo "${v#go}"; })"
go mod tidy
go get -u ./

go build . && echo -e "\033[32mBuild: ASnake\033[0m" || echo -e "\033[31mFailed: ASnake\033[0m"
