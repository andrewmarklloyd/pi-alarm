package util

type Config struct {
	ClientID        string
	ClientSecret    string
	RedirectURL     string
	AuthorizedUsers string
	Pin             int
	Debug           bool
}

type StatusPageData struct {
	Status string
}
