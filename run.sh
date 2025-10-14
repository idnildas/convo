#!/bin/zsh
# Script to run the main entry point of the Go application

golang_main="cmd/api/main.go"

echo "Running Go application..."
go run "$golang_main"
