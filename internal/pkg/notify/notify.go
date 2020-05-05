package notify

import (
	"fmt"
	"log"
)

func SendMessage(state string) {
	log.Println(fmt.Sprintf("State changed, current state: %s", state))
}
