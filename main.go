package main

import (
	"fmt"
	"os"
	"os/signal"
	
	"syscall"
	"net/http"
	"github.com/robfig/cron/v3"
	"github.com/stianeikeland/go-rpio"
)

var testmode = false
var cronLib *cron.Cron
var status rpio.State

func main() {
	pinNumber := 18
	pin := rpio.Pin(pinNumber)

	err := rpio.Open()
	if err != nil {
		fmt.Println("unable to open gpio", err.Error())
		fmt.Println("running in test mode")
		testmode = true
	}

	if !testmode {
		pin.Input()
		pin.Pull(rpio.PullUp)
	}

	cronLib = cron.New()
	cronLib.AddFunc("@every 0h0m1s", func() {
		if !testmode {
			status = pin.Read()
		} else {
			status = 1
		}
		fmt.Println("status:", status)
	})
	cronLib.Start()

	fmt.Println("creating channel")
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cleanup(pin, 18)
		os.Exit(1)
	}()

	fmt.Println("Setting up http handler")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello")
	})
	http.ListenAndServe("0.0.0.0:8080", nil)
}

func cleanup(pin rpio.Pin, pinNumber int) {
	fmt.Println("Cleaning up pin", pinNumber)
	rpio.Close()
}
