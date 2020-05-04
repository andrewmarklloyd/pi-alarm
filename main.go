package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	gpioLib "github.com/andrewmarklloyd/pi-alarm/internal/pkg/gpio"
	"github.com/andrewmarklloyd/pi-alarm/internal/pkg/util"
	"github.com/andrewmarklloyd/pi-alarm/internal/pkg/web"
)

const (
	defaultPin  = 18
	PRIVATE_DIR = "/private/"
)

var config *util.Config
var gpio gpioLib.GPIO

func main() {
	const address = "0.0.0.0:8080"

	debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	var pinNum int
	pinNum, err := strconv.Atoi(os.Getenv("GPIO_PIN"))
	if err != nil {
		log.Printf("Failed to parse GPIO_PIN env var, using default %d", defaultPin)
		pinNum = defaultPin
	}
	config = &util.Config{
		ClientID:        os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret:    os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:     os.Getenv("REDIRECT_URL"),
		AuthorizedUsers: os.Getenv("AUTHORIZED_USERS"),
		Pin:             pinNum,
		Debug:           debug,
	}

	if config.ClientID == "" {
		log.Fatal("Missing Google Client ID")
	}
	if config.ClientSecret == "" {
		log.Fatal("Missing Google Client Secret")
	}
	if config.RedirectURL == "" {
		log.Fatal("Missing Google Redirect URL")
	}
	if config.AuthorizedUsers == "" {
		log.Fatal("Missing Authorized Users")
	}

	gpio = gpioLib.GPIO{}
	err = gpio.SetupGPIO(config.Pin)
	server := web.NewServer(config, statusHandler)

	log.Println("Creating channel to cleanup GPIO pins")
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		gpio.Cleanup()
		os.Exit(1)
	}()

	log.Printf("Starting Server listening on %s\n", address)

	err = http.ListenAndServe(address, server)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// statusHandler shows protected user content.
func statusHandler(w http.ResponseWriter, req *http.Request) {
	tmpl := template.Must(template.ParseFiles(fmt.Sprintf(".%sstatus.html", PRIVATE_DIR)))
	data := util.StatusPageData{
		Status: gpio.CurrentStatus(),
	}
	tmpl.Execute(w, data)
}
