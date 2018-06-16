package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	hub "github.com/GregoryDosh/game-server/hub"
	hi "github.com/GregoryDosh/game-server/hub/hubinterfaces"
	"github.com/GregoryDosh/game-server/hub/moose"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
	cli "github.com/urfave/cli"
)

// Global websocket connection parameters
const (
	pongWait       = 60 * time.Second
	pingPeriod     = pongWait * 9 / 10
	writeWait      = 10 * time.Second
	maxMessageSize = 1024
)

var (
	origin   string
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			if origin == "*" {
				return true
			}
			if origin == r.Header.Get("Origin") {
				return true
			}
			return false
		},
	}
)

// Global cookie parameters
var sc *securecookie.SecureCookie

func main() {
	app := cli.NewApp()
	app.Name = "sh"
	app.Usage = "serve up games through a websocket connections"
	app.Version = "0.1"
	app.Action = appEntry
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "host",
			Usage:  "Hostname to listen on",
			Value:  "localhost",
			EnvVar: "LISTEN_HOST",
		},
		cli.IntFlag{
			Name:   "port",
			Usage:  "TCP `port` to listen on",
			Value:  9999,
			EnvVar: "LISTEN_PORT",
		},
		cli.StringFlag{
			Name:   "hash-key",
			Usage:  "Hash key used for secure cookies",
			EnvVar: "HASH_KEY",
		},
		cli.StringFlag{
			Name:   "block-key",
			Usage:  "Block key used for secure cookies",
			EnvVar: "BLOCK_KEY",
		},
		cli.StringFlag{
			Name:        "origin",
			Usage:       "Sets the allowable origin",
			Value:       "*",
			EnvVar:      "ORIGIN",
			Destination: &origin,
		},
		cli.StringFlag{
			Name:   "log-level,l",
			Usage:  "Log `level` for output",
			Value:  "debug",
			EnvVar: "LOG_LEVEL",
		},
		cli.StringFlag{
			Name:  "cpuprofile",
			Value: "",
		},
		cli.StringFlag{
			Name:  "memprofile",
			Value: "",
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Error(err)
	}
}

func appEntry(c *cli.Context) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	host := c.String("host")
	port := c.Int("port")
	cpuprofile := c.String("cpuprofile")
	memprofile := c.String("memprofile")
	hashKey := []byte(c.String("hash-key"))
	blockKey := []byte(c.String("block-key"))
	switch strings.ToLower(c.String("log-level")) {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	}

	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	if len(hashKey) == 0 {
		log.Debug("Generating hashKey")
		hashKey = securecookie.GenerateRandomKey(32)
	}

	switch len(blockKey) {
	case 16, 24, 32:
	case 0:
		log.Debug("encryption disabled")
		blockKey = nil
	default:
		log.Debug("Invalid blockKey size using generated blockKey")
		blockKey = []byte(securecookie.GenerateRandomKey(32))
	}

	sc = securecookie.New(hashKey, blockKey)

	go httpRouteHandler(host, port)

	<-stop

	log.Info("shutting down")

	if memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}
}

func userCookieHandler(w http.ResponseWriter, r *http.Request, cookieName string) (string, bool) {
	// If the cookie is valid, let em through and return the key-value pairs & true for being okay
	if cookie, err := r.Cookie(cookieName); err == nil {
		u := ""
		if err = sc.Decode(cookieName, cookie.Value, &u); err == nil {
			return u, true
		}
	}
	// If here, we're assuming cookie doesn't exist or isn't valid, so give them a UUID to use and return it.
	u := uuid.Must(uuid.NewV4()).String()
	encoded, err := sc.Encode(cookieName, u)
	if err != nil {
		log.Error(err)
		return "", false
	}
	cookie := &http.Cookie{
		Name:  cookieName,
		Value: encoded,
	}
	http.SetCookie(w, cookie)
	return u, false
}

func httpRouteHandler(host string, port int) {
	h := hub.New()

	g1 := &moose.GameSecretMoose{GameName: "Lunchtime Brawl"}
	g2 := &moose.GameSecretMoose{GameName: "HH Checkn"}
	for _, p := range []*hi.LobbyPlayer{&hi.LobbyPlayer{Name: "Me"}, &hi.LobbyPlayer{Name: "You"}, &hi.LobbyPlayer{Name: "Them"}, &hi.LobbyPlayer{Name: "P4"}, &hi.LobbyPlayer{Name: "P5"}, &hi.LobbyPlayer{Name: "P6"}, &hi.LobbyPlayer{Name: "P7"}, &hi.LobbyPlayer{Name: "P8"}} {
		if _, err := g1.AddPlayer(p); err != nil {
			log.Error(err)
		}
		g1.PlayerEvent(p, &hi.MessageFromPlayer{
			Type: "ToggleReady",
		})
	}
	if err := g1.StartGame(); err != nil {
		log.Error(err)
	}
	g1.EndGame()
	if _, err := h.AddGame(g1); err != nil {
		log.Error(err)
	}
	if _, err := h.AddGame(g2); err != nil {
		log.Error(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, validUser := userCookieHandler(w, r, "userID")
		if !validUser {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocketHandler(w, r, h)
	})
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func websocketHandler(w http.ResponseWriter, r *http.Request, h *hub.Hub) {
	userID, validUser := userCookieHandler(w, r, "userID")
	if !validUser {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Upgrade normal http request into a websocket session
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Add some default websocket parameters
	ws.SetReadLimit(maxMessageSize)
	if err := ws.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Error(err)
		return
	}
	ws.SetPongHandler(func(string) error {
		return ws.SetReadDeadline(time.Now().Add(pongWait))
	})

	// Retrieve/connect websocket to player
	s, err := h.ConnectSession(userID, ws, pingPeriod)
	if err != nil {
		log.Error(err)
		return
	}
	h.UpdateGameList(s)
}
