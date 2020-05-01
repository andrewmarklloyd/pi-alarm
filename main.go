package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	gmux "github.com/gorilla/mux"
	"strconv"
	"os"
	"os/signal"
	"syscall"
	"strings"
	"github.com/stianeikeland/go-rpio"
	"github.com/dghubble/gologin/v2"
	"github.com/dghubble/gologin/v2/google"
	"github.com/dghubble/sessions"
	"golang.org/x/oauth2"
	googleOAuth2 "golang.org/x/oauth2/google"
)

const (
	sessionName    = "pi-alarm"
	sessionSecret  = "example cookie signing secret"
	sessionUserKey = "googleID"
	defaultPin     = 18
	STATIC_DIR     = "/static/"
)

// sessionStore encodes and decodes session data stored in signed cookies
var sessionStore = sessions.NewCookieStore([]byte(sessionSecret), nil)

var testmode = false
var pin rpio.Pin
var config *Config

type Config struct {
	ClientID        string
	ClientSecret    string
	RedirectURL     string
	AuthorizedUsers string
	Pin             int
	Debug           bool
}

func main() {
	const address = "0.0.0.0:8080"

	debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	var pin int
	pin, err := strconv.Atoi(os.Getenv("GPIO_PIN"))
  if err != nil {
    log.Printf("Failed to parse GPIO_PIN env var, using default %d", defaultPin)
		pin = defaultPin
  }
	config = &Config{
		ClientID:        os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret:    os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:     os.Getenv("REDIRECT_URL"),
		AuthorizedUsers: os.Getenv("AUTHORIZED_USERS"),
		Pin:             pin,
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

	setupGPIO(config.Pin)

	log.Println("Creating channel to cleanup GPIO pins")
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cleanup(config.Pin)
		os.Exit(1)
	}()

	log.Printf("Starting Server listening on %s\n", address)

	err = http.ListenAndServe(address, New(config))
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// New returns a new ServeMux with app routes.
func New(config *Config) *gmux.Router {
	router := gmux.NewRouter().StrictSlash(true)
	router.
    PathPrefix(STATIC_DIR).
    Handler(http.StripPrefix(STATIC_DIR, http.FileServer(http.Dir("."+STATIC_DIR))))

	router.HandleFunc("/", welcomeHandler)
	router.Handle("/status", requireLogin(http.HandlerFunc(statusHandler)))
	router.HandleFunc("/logout", logoutHandler)
	// 1. Register Login and Callback handlers
	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Endpoint:     googleOAuth2.Endpoint,
		Scopes:       []string{"profile", "email"},
	}
	// state param cookies require HTTPS by default; disable for localhost development
	stateConfig := gologin.DebugOnlyCookieConfig
	router.Handle("/google/login", google.StateHandler(stateConfig, google.LoginHandler(oauth2Config, nil)))
	router.Handle("/google/callback", google.StateHandler(stateConfig, google.CallbackHandler(oauth2Config, issueSession(), nil)))
	return router
}

// issueSession issues a cookie session after successful Google login
func issueSession() http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		googleUser, err := google.UserFromContext(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !strings.Contains(config.AuthorizedUsers, googleUser.Email) {
			http.Redirect(w, req, "/static/error.html", http.StatusFound)
			return
		}
		// 2. Implement a success handler to issue some form of session
		session := sessionStore.New(sessionName)
		session.Values[sessionUserKey] = googleUser.Id
		session.Save(w)
		http.Redirect(w, req, "/status", http.StatusFound)
	}
	return http.HandlerFunc(fn)
}

// welcomeHandler shows a welcome message and login button.
func welcomeHandler(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	if isAuthenticated(req) {
		http.Redirect(w, req, "/status", http.StatusFound)
		return
	}

	page, _ := ioutil.ReadFile("./static/index.html")
	fmt.Fprintf(w, string(page))
}

// statusHandler shows protected user content.
func statusHandler(w http.ResponseWriter, req *http.Request) {
	message := fmt.Sprintf(`<p>Status: %s</p><form action="/logout" method="post"><input type="submit" value="Logout"></form>`, currentStatus())
	fmt.Fprint(w, message)
}

// logoutHandler destroys the session on POSTs and redirects to home.
func logoutHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		sessionStore.Destroy(w, sessionName)
	}
	http.Redirect(w, req, "/", http.StatusFound)
}

// requireLogin redirects unauthenticated users to the login route.
func requireLogin(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		if !isAuthenticated(req) {
			http.Redirect(w, req, "/", http.StatusFound)
			return
		}
		next.ServeHTTP(w, req)
	}
	return http.HandlerFunc(fn)
}

// isAuthenticated returns true if the user has a signed session cookie.
func isAuthenticated(req *http.Request) bool {
	if _, err := sessionStore.Get(req, sessionName); err == nil {
		return true
	}
	return false
}

func setupGPIO(pinNumber int) {
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
}

func cleanup(pinNumber int) {
	fmt.Println("Cleaning up pin", pinNumber)
	rpio.Close()
}

func currentStatus() string {
	if !testmode {
		log.Println("pin read:", pin.Read())
		return strconv.Itoa(int(pin.Read()))
	}
	return strconv.Itoa(0)
}
