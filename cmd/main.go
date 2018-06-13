package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	hub "github.com/GregoryDosh/game-server/hub"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
	cli "github.com/urfave/cli"
)

// Global websocket connection parameters
const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = pongWait * 9 / 10
	maxMessageSize = 1024
)

var (
	origin   string
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
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
	app.Usage = "server up games through a websocket connections"
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
	}
	if err := app.Run(os.Args); err != nil {
		log.Error(err)
	}
}

func appEntry(c *cli.Context) {
	host := c.String("host")
	port := c.Int("port")
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

	httpRouteHandler(host, port)
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

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	h.ConnectSession(userID, ws)
	// p := &Player{
	// 	hub:      h,
	// 	ws:       ws,
	// 	toPlayer: make(chan *MessageToPlayer, 256),
	// }
	// h.join <- p

	// go p.toPlayerHandler()
	// go p.fromPlayerHandler()
}
