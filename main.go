package main

import (
	"fmt"
	"os"
	"os/signal"
	"io/ioutil"
	"syscall"
	"net/http"
	"text/template"
	"github.com/robfig/cron/v3"
	"github.com/stianeikeland/go-rpio"
	"github.com/andrewmarklloyd/pi-alarm/internal/pkg/socket"
)

type config struct {
	Server struct {
		Pin        int    `yaml:"pin"`
		Debug      bool   `yaml:"debug"`
		AutoUpdate bool   `yaml:"autoUpdate`
	} `yaml:"server"`
}

type HomePageData struct {
	Version       string
	LatestVersion string
	Debug         bool
}

var cfg config
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
		// fmt.Println("status:", status)
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

	version, err := ioutil.ReadFile("static/version")
	if err != nil {
		fmt.Println("unable to open version", err)
		os.Exit(1)
	}

	fmt.Println("Setting up http handlers")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[1:]

		if path == "" {
			latestVersion, err := ioutil.ReadFile("static/latestVersion")
			if err != nil || len(latestVersion) == 0 {
				latestVersion = version
			}
			tmpl := template.Must(template.ParseFiles("./static/index.html"))
			data := HomePageData{
				Version:       string(version),
				LatestVersion: string(latestVersion),
				Debug:         cfg.Server.Debug,
			}
			tmpl.Execute(w, data)
		} else {
			if fileExists(path) {
				d, _ := ioutil.ReadFile(string(path))
				w.Write(d)
			} else {
				// fmt.Println(path)
				http.NotFound(w, r)
			}
		}
	})
	socket.Init()
	http.ListenAndServe("0.0.0.0:8080", nil)
}

func cleanup(pin rpio.Pin, pinNumber int) {
	fmt.Println("Cleaning up pin", pinNumber)
	rpio.Close()
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
