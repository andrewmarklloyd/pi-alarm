#!/bin/bash

build() {
  GOOS=linux GOARCH=arm GOARM=5 go build -o pi-alarm main.go
}

if [[ ${1} == 'build' ]]; then
  build || exit 1
fi
