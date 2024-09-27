#!/bin/bash

go mod edit -go "$(go version | { read -r _ _ v _; echo "${v#go}"; })"
go mod tidy
go get -u ./
