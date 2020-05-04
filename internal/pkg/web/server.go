package web

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"text/template"

	"github.com/andrewmarklloyd/pi-alarm/internal/pkg/gpio"
	"github.com/andrewmarklloyd/pi-alarm/internal/pkg/util"
	"github.com/dghubble/gologin/v2"
	"github.com/dghubble/gologin/v2/google"
	"github.com/dghubble/sessions"
	gmux "github.com/gorilla/mux"
	"golang.org/x/oauth2"
	googleOAuth2 "golang.org/x/oauth2/google"
)

const (
	sessionName     = "pi-alarm"
	sessionSecret   = "example cookie signing secret"
	sessionUserKey  = "googleID"
	defaultPin      = 18
	PUBLIC_DIR      = "/public/"
	PRIVATE_DIR     = "/private/"
	STATUS_ENDPOINT = "/status"
)

var config *util.Config

// sessionStore encodes and decodes session data stored in signed cookies
var sessionStore = sessions.NewCookieStore([]byte(sessionSecret), nil)

// New returns a new ServeMux with app routes.
func NewServer(config *util.Config) *gmux.Router {
	router := gmux.NewRouter().StrictSlash(true)
	router.
		PathPrefix(PUBLIC_DIR).
		Handler(http.StripPrefix(PUBLIC_DIR, http.FileServer(http.Dir("."+PUBLIC_DIR))))

	router.HandleFunc("/", welcomeHandler)
	router.Handle(STATUS_ENDPOINT, requireLogin(http.HandlerFunc(statusHandler)))
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
			http.Redirect(w, req, fmt.Sprintf("%serror.html", PUBLIC_DIR), http.StatusFound)
			return
		}
		// 2. Implement a success handler to issue some form of session
		session := sessionStore.New(sessionName)
		session.Values[sessionUserKey] = googleUser.Id
		session.Save(w)
		http.Redirect(w, req, STATUS_ENDPOINT, http.StatusFound)
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
		http.Redirect(w, req, STATUS_ENDPOINT, http.StatusFound)
		return
	}

	page, _ := ioutil.ReadFile(fmt.Sprintf(".%sindex.html", PUBLIC_DIR))
	fmt.Fprintf(w, string(page))
}

// statusHandler shows protected user content.
func statusHandler(w http.ResponseWriter, req *http.Request) {
	tmpl := template.Must(template.ParseFiles(fmt.Sprintf(".%sstatus.html", PRIVATE_DIR)))
	data := util.StatusPageData{
		Status: gpio.CurrentStatus(),
	}
	tmpl.Execute(w, data)
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
