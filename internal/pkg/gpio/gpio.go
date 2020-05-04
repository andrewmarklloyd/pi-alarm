package gpio

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/stianeikeland/go-rpio"
)

const (
	OPEN    = "OPEN"
	CLOSED  = "CLOSED"
	UNKNOWN = "UNKNOWN"
)

var testmode = false
var pin rpio.Pin

func SetupGPIO(pinNumber int) (rpio.Pin, error) {
	pin = rpio.Pin(pinNumber)

	err := rpio.Open()
	if err != nil {
		log.Println(fmt.Sprintf("Unable to open gpio: %s, continuing but running in test mode.", err.Error()))
		testmode = true
	}

	if !testmode {
		pin.Input()
		pin.Pull(rpio.PullUp)
	}

	return pin, nil
}

func cleanup(pinNumber int) {
	fmt.Println("Cleaning up pin", pinNumber)
	rpio.Close()
}

func CurrentStatus() string {
	var pinState int
	if testmode {
		pinState = rand.Intn(2)
	} else {
		pinState = int(pin.Read())
	}

	if pinState == 0 {
		return CLOSED
	} else if pinState == 1 {
		return OPEN
	}
	return UNKNOWN
}
