package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
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
	privateDir             = "/private/"
	// Maximum message size allowed from peer.
	maxMessageSize = 8192
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second
	open     = "OPEN"
	closed   = "CLOSED"
)

var upgrader = websocket.Upgrader{}

var config *util.Config
var gpio gpioLib.GPIO
var cronLib *cron.Cron
var messenger notify.Messenger
var testMessageMode = false
var version []byte
var maxDoorOpenedTime time.Duration

type System struct {
	Operation string
}

type Arming struct {
	Armed bool `json:"armed"`
}

type Event struct {
	Message string
}

type AppInfo struct {
	TagName string `json:"tag_name"`
}

func main() {
	var address string
	if macMode() {
		address = "localhost:8080"
		maxDoorOpenedTime = 5 * time.Second
	} else {
		address = "0.0.0.0:8080"
		maxDoorOpenedTime = 5 * time.Minute
	}

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
	version, err = ioutil.ReadFile("public/version")
	if err != nil {
		log.Println("Unable to open version", err)
		os.Exit(1)
	}

	gpio = gpioLib.GPIO{}
	err = gpio.SetupGPIO(config.Pin)
	server := web.NewServer(config, statusHandler, websocketHandler, systemHandler, alertNotifyHandler)
	messenger = notify.Messenger{
		AccountSID: config.TwilioAccountSID,
		AuthToken:  config.TwilioAuthToken,
	}

	cronLib = cron.New()
	configureStateChanged(config.StatusInterval)
	configureOpenAlert(config.StatusInterval)
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

func systemHandler(w http.ResponseWriter, req *http.Request) {
	var system System
	err := json.NewDecoder(req.Body).Decode(&system)
	if err != nil {
		http.Error(w, "Error parsing operation", http.StatusBadRequest)
		return
	}

	var args = []string{}
	command := "sudo"
	switch system.Operation {
	case "shutdown":
		args = []string{"shutdown", "now"}
		fmt.Fprintf(w, "shutting down")
	case "reboot":
		args = []string{"reboot", "now"}
		fmt.Fprintf(w, "rebooting")
	case "check-updates":
		checkForUpdates()
		fmt.Fprintf(w, "checking for updates")
	default:
		fmt.Fprintf(w, "command not recognized")
	}
	if command != "" && !macMode() {
		log.Printf("Running command: %s %s\n", command, args)
		cmd := exec.Command(command, args...)
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Start()
		if err != nil {
			log.Println("Failed to initiate command:", err)
		} else {
			fmt.Printf("Command output: %q\n", out.String())
		}
	}
}

func alertNotifyHandler(w http.ResponseWriter, req *http.Request) {
	var n notify.NotifyEvent
	err := json.NewDecoder(req.Body).Decode(&n)
	if err != nil {
		http.Error(w, "Error parsing operation", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "ok")
}

func checkForUpdates() {
	log.Println("Checking for updates")
	resp, err := http.Get("https://api.github.com/repos/andrewmarklloyd/pi-alarm/releases/latest")
	if err != nil {
		log.Println(err)
	}
	var info AppInfo
	err = json.NewDecoder(resp.Body).Decode(&info)
	if err != nil {
		log.Println("Error parsing version:", err)
	} else {
		log.Println("Writing latestVersion to file")
		latestVersion := []byte(info.TagName)
		err = ioutil.WriteFile("./public/latestVersion", latestVersion, 0644)
		if err != nil {
			fmt.Println(err)
		}

		version, err := ioutil.ReadFile("public/version")
		if err != nil {
			fmt.Println("unable to open version", err)
			os.Exit(1)
		}
		if info.TagName != string(version) && !macMode() {
			log.Println("Updating software version")
			cmd := exec.Command("/home/pi/install/update.sh")
			var out bytes.Buffer
			cmd.Stdout = &out
			err := cmd.Start()
			if err != nil {
				log.Println("Failed to initiate command:", err)
			}
		}
	}
}

func websocketHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("New websocket connection")
	ws, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}

	defer ws.Close()
	ws.SetReadLimit(maxMessageSize)
	sendState(ws)
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
			sendState(ws)
		} else {
			log.Println("recv: " + string(message))
		}
	}
}

func sendState(ws *websocket.Conn) error {
	state, err := util.ReadState()
	if err != nil {
		log.Println("Error getting armed status: ", err)
		return err
	}
	ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("{\"type\":\"armed\",\"value\":%v}", state.Armed)))
	ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("{\"type\":\"status\",\"value\":\"%v\"}", state.LastKnownStatus)))
	return nil
}

// statusHandler shows protected user content.
func statusHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		tmpl := template.Must(template.ParseFiles(fmt.Sprintf(".%sstatus.html", privateDir)))

		latestVersion, err := ioutil.ReadFile("public/latestVersion")
		if err != nil || len(latestVersion) == 0 {
			latestVersion = version
		}

		data := util.StatusPageData{
			Status:        gpio.CurrentStatus(),
			Version:       string(version),
			LatestVersion: string(latestVersion),
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
			if state.LastKnownStatus == closed && currentStatus == open {
				state.FirstReportedOpenTime = time.Now().Format(time.RFC3339)
				log.Println(fmt.Sprintf("State changed, current state: %s", currentStatus))
				if !testMessageMode && state.Armed {
					messenger.SendMessage(fmt.Sprintf("Door is %s", currentStatus))
				}
			} else if state.LastKnownStatus == open && currentStatus == closed {
				state.AlertNotified = false
				state.FirstReportedOpenTime = ""
			} else {
				// intentionally do nothing
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
		if state.FirstReportedOpenTime != "" {
			firstReportedOpenTime, _ := time.Parse(time.RFC3339, state.FirstReportedOpenTime)
			now := time.Now()
			maxTimeSinceDoorOpened := now.Add(-maxDoorOpenedTime)
			if firstReportedOpenTime.Before(maxTimeSinceDoorOpened) && !state.AlertNotified {
				message := fmt.Sprintf("Door opened for longer than %s", maxDoorOpenedTime)
				if testMessageMode {
					log.Println(message)
				} else {
					messenger.SendMessage(message)
				}
				state.AlertNotified = true
				util.WriteState(state)
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

func macMode() bool {
	return runtime.GOOS == "darwin"
}
