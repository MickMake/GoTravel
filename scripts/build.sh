#!/bin/bash

go mod tidy
go test ./...
go build -o ~/go/bin/GoTravel .

