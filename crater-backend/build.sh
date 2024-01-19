#!/bin/bash
# go mod tidy && go mod vendor
export GIN_MODE=release && go build -mod=vendor -o bin/controller main.go
