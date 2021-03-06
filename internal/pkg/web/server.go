package web

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/andrewmarklloyd/pi-alarm/internal/pkg/util"
	"github.com/dghubble/gologin/v2"
	"github.com/dghubble/gologin/v2/google"
	"github.com/dghubble/sessions"
	gmux "github.com/gorilla/mux"
	"golang.org/x/oauth2"
	googleOAuth2 "golang.org/x/oauth2/google"
)

const (
	sessionName    = "pi-alarm"
	sessionUserKey = "googleID"
	defaultPin     = 18
	publicDir      = "/public/"
	statusEndpoint = "/status"
	post           = "post"
)

var config *util.Config

// sessionStore encodes and decodes session data stored in signed cookies
var sessionStore *sessions.CookieStore

// NewServer returns a new ServeMux with app routes.
func NewServer(utilConfig *util.Config, statusHandler http.HandlerFunc, websocketHandler http.HandlerFunc, systemHandler http.HandlerFunc, alertNotifyHandler http.HandlerFunc) *gmux.Router {
	config = utilConfig
	sessionStore = sessions.NewCookieStore([]byte(config.SessionSecret), nil)
	router := gmux.NewRouter().StrictSlash(true)
	router.
		PathPrefix(publicDir).
		Handler(http.StripPrefix(publicDir, http.FileServer(http.Dir("."+publicDir))))

	router.HandleFunc("/", welcomeHandler)
	router.Handle(statusEndpoint, requireLogin(http.HandlerFunc(statusHandler))).Methods("GET", post)
	router.Handle("/ws", requireLogin(http.HandlerFunc(websocketHandler)))
	router.Handle("/system", requireLogin(http.HandlerFunc(systemHandler))).Methods(post)
	router.Handle("/notify", requireLogin(http.HandlerFunc(alertNotifyHandler))).Methods(post)
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
			http.Redirect(w, req, fmt.Sprintf("%serror.html", publicDir), http.StatusFound)
			return
		}
		// 2. Implement a success handler to issue some form of session
		session := sessionStore.New(sessionName)
		session.Values[sessionUserKey] = googleUser.Id
		session.Save(w)
		http.Redirect(w, req, statusEndpoint, http.StatusFound)
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
		http.Redirect(w, req, statusEndpoint, http.StatusFound)
		return
	}

	page, _ := ioutil.ReadFile(fmt.Sprintf(".%sindex.html", publicDir))
	fmt.Fprintf(w, string(page))
}

// logoutHandler destroys the session on posts and redirects to home.
func logoutHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method == post {
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
