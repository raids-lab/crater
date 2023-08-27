#!/bin/bash
# go mod tidy && go mod vendor
go build -mod=vendor -o bin/controller main.go
