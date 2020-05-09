package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	gpioLib "github.com/andrewmarklloyd/pi-alarm/internal/pkg/gpio"
	"github.com/andrewmarklloyd/pi-alarm/internal/pkg/notify"
	"github.com/andrewmarklloyd/pi-alarm/internal/pkg/util"
	"github.com/andrewmarklloyd/pi-alarm/internal/pkg/web"
	"github.com/gorilla/websocket"
	"github.com/robfig/cron/v3"
)

const (
	defaultPin             = 18
	defaultIntervalSeconds = 10
	PRIVATE_DIR            = "/private/"
	// Maximum message size allowed from peer.
	maxMessageSize = 8192
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second
)

var testmode = false
var upgrader = websocket.Upgrader{}

var config *util.Config
var gpio gpioLib.GPIO
var cronLib *cron.Cron
var messenger notify.Messenger
var testMessageMode bool = false

type Arming struct {
	Armed bool `json:"armed"`
}

type Event struct {
	Message string
}

func main() {
	const address = "0.0.0.0:8080"

	debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	var pinNum int
	pinNum, err := strconv.Atoi(os.Getenv("GPIO_PIN"))
	if err != nil {
		log.Printf("Failed to parse GPIO_PIN env var, using default %d", defaultPin)
		pinNum = defaultPin
	}
	var statusInterval int
	statusInterval, err = strconv.Atoi(os.Getenv("STATUS_INTERVAL"))
	if err != nil {
		log.Printf("Failed to parse STATUS_INTERVAL env var, using default %d", defaultIntervalSeconds)
		statusInterval = defaultIntervalSeconds
	}
	config = &util.Config{
		ClientID:         os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret:     os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:      os.Getenv("REDIRECT_URL"),
		AuthorizedUsers:  os.Getenv("AUTHORIZED_USERS"),
		Pin:              pinNum,
		StatusInterval:   statusInterval,
		Debug:            debug,
		TwilioAccountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
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
	if config.TwilioAccountSID == "" && config.TwilioAuthToken == "" {
		log.Println("Twilio auth env vars not set, running in testMessageMode")
		testMessageMode = true
	}
	if config.AuthorizedUsers == "" {
		log.Fatal("Missing Authorized Users")
	}

	gpio = gpioLib.GPIO{}
	err = gpio.SetupGPIO(config.Pin)
	server := web.NewServer(config, statusHandler, websocketHandler)
	messenger = notify.Messenger{
		AccountSID: config.TwilioAccountSID,
		AuthToken:  config.TwilioAuthToken,
	}

	cronLib = cron.New()
	// configureStateChanged(config.StatusInterval)
	// configureOpenAlert(config.StatusInterval)
	cronLib.Start()

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

func websocketHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("*****")
	ws, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}

	defer ws.Close()
	ws.SetReadLimit(maxMessageSize)
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			break
		}
		var event Event
		err = json.Unmarshal(message, &event)
		if err != nil {
			log.Println("Error unmarshalling json: ", err)
			break
		}
		if event.Message == "ping" {
			ws.WriteMessage(websocket.TextMessage, []byte("{\"event\":\"pong\"}"))
		} else {
			log.Println("recv: " + string(message))
		}
	}
}

// statusHandler shows protected user content.
func statusHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		tmpl := template.Must(template.ParseFiles(fmt.Sprintf(".%sstatus.html", PRIVATE_DIR)))
		data := util.StatusPageData{
			Status: gpio.CurrentStatus(),
		}
		tmpl.Execute(w, data)
	} else if req.Method == "POST" {
		var arming Arming
		err := json.NewDecoder(req.Body).Decode(&arming)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		setArmed(arming.Armed)
		fmt.Fprintf(w, "{\"armed\": %+v}", arming.Armed)
	} else {

	}
}

func configureStateChanged(statusInterval int) {
	cronLib.AddFunc(fmt.Sprintf("@every %ds", statusInterval), func() {
		state, err := util.ReadState()
		if err != nil {
			log.Println("Error reading state file: ", err)
		} else {
			currentStatus := gpio.CurrentStatus()
			if state.LastKnownStatus != currentStatus && state.Armed {
				if !testMessageMode {
					messenger.SendMessage(fmt.Sprintf("Door is %s", currentStatus))
				} else {
					log.Println(fmt.Sprintf("State changed, current state: %s", state.LastKnownStatus))
				}
			}
			state.LastKnownStatus = currentStatus
			util.WriteState(state)
		}
	})
}

func configureOpenAlert(statusInterval int) {
	cronLib.AddFunc(fmt.Sprintf("@every %ds", statusInterval), func() {
		state, err := util.ReadState()
		if err != nil {
			log.Println(fmt.Sprintf("Error getting armed status: %s", err))
			return
		}
		if state.LastKnownStatus == "OPEN" && state.Armed {
			if testMessageMode {
				log.Printf("Alert, door is %s ", state.LastKnownStatus)
			} else {
				messenger.SendMessage(fmt.Sprintf("Alert, door is %s", state.LastKnownStatus))
			}
		}
	})
}

func setArmed(armed bool) {
	state, err := util.ReadState()
	if err != nil {
		log.Println("Error reading state file: ", err)
	} else {
		state.Armed = armed
		log.Printf("Setting armed to %v", state.Armed)
		util.WriteState(state)
	}
}

func isArmed() (bool, error) {
	state, err := util.ReadState()
	if err != nil {
		return false, err
	}
	return state.Armed, nil
}
