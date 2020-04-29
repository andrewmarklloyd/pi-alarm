# Pi Alarm

[![Build Status](https://travis-ci.org/andrewmarklloyd/pi-alarm.svg?branch=master)](https://travis-ci.org/andrewmarklloyd/pi-alarm)

Proof of concept for using a magnetic sensor to detect open doors, windows, or even garage doors.

### One Line Install
When a new [release](https://github.com/andrewmarklloyd/pi-alarm/releases) is created, Travis CI will build the binary and attach it to the release. To install the latest release on a Raspberry Pi with a single line command, run the following:
```
bash <(curl -s -H 'Cache-Control: no-cache' https://raw.githubusercontent.com/andrewmarklloyd/pi-alarm/master/install/install.sh)
```

### Developing Locally
Requires Go 1.13.1 to build the project.
```
# run the program
go run main.go

# build an executable
go build -o pi-alarm main.go
```
