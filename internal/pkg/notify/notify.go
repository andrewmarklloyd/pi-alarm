package notify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Messenger struct {
	AccountSID string
	AuthToken  string
}

func (m Messenger) SendMessage(state string) {
	urlStr := "https://api.twilio.com/2010-04-01/Accounts/" + m.AccountSID + "/Messages.json"

	// Pack up the data for our message
	msgData := url.Values{}
	msgData.Set("To", "+12534484393")
	msgData.Set("From", "+12025190922")
	msgData.Set("Body", state)
	msgDataReader := *strings.NewReader(msgData.Encode())

	// Create HTTP request client
	client := &http.Client{}
	req, _ := http.NewRequest("POST", urlStr, &msgDataReader)
	req.SetBasicAuth(m.AccountSID, m.AuthToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Make HTTP POST request and return message SID
	resp, _ := client.Do(req)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var data map[string]interface{}
		decoder := json.NewDecoder(resp.Body)
		err := decoder.Decode(&data)
		if err == nil {
			fmt.Println(data["sid"])
		}
	} else {
		fmt.Println(resp.Status)
	}
}
