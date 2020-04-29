package socket

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/robfig/cron/v3"
	"github.com/yryz/ds18b20"
)

var upgrader = websocket.Upgrader{}
var sensor string

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 8192

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Time to wait before force close on connection.
	closeGracePeriod = 10 * time.Second
)

func serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}

	defer ws.Close()

	cronLib := cron.New()
	cronLib.AddFunc("@every 1s", func() {

		ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%f", currentTemp())))
	})
	cronLib.Start()

	ws.SetReadLimit(maxMessageSize)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			break
		}
		fmt.Println("recv: " + string(message))
	}
}

func Init() {
	fmt.Println("setting up websockets")
	sensors, err := ds18b20.Sensors()
	if err != nil {
		fmt.Println("Error setting up sensors:", err)
	} else {
		sensor = sensors[0]
		currentTemp()
	}

	http.HandleFunc("/ws", serveWs)
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}

func currentTemp() float64 {
	celcius, _ := ds18b20.Temperature(sensor)
	temp := (celcius * (9 / 5)) + 32
	return temp
}
