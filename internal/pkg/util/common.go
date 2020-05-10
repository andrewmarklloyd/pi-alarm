package util

type Config struct {
	ClientID        string
	ClientSecret    string
	RedirectURL     string
	AuthorizedUsers string
	Pin             int
	Debug           bool
	// StatusInterval is the interval of seconds to check the sensor status
	StatusInterval   int
	TwilioAccountSID string
	TwilioAuthToken  string
}

type StatusPageData struct {
	Status        string
	Version       string
	LatestVersion string
}
