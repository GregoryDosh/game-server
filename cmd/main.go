package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	hub "github.com/GregoryDosh/game-server/hub"
	moose "github.com/GregoryDosh/game-server/hub/moose"
	log "github.com/Sirupsen/logrus"
	cli "github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "sh"
	app.Usage = "server up games through a websocket connections"
	app.Version = "0.1"
	app.Action = appEntry
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:   "port",
			Usage:  "TCP `port` to listen on",
			Value:  9999,
			EnvVar: "LISTEN_PORT",
		},
		cli.StringFlag{
			Name:   "host",
			Usage:  "Hostname to listen on",
			Value:  "localhost",
			EnvVar: "LISTEN_HOST",
		},
		cli.StringFlag{
			Name:   "log-level,l",
			Usage:  "Log `level` for output",
			EnvVar: "LOG_LEVEL",
			Value:  "debug",
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Error(err)
	}
}

func appEntry(c *cli.Context) {
	port := c.Int("port")
	host := c.String("host")
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

	httpRouteHandler(host, port)
}

func httpRouteHandler(host string, port int) {
	h := hub.New()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error(err)
			return
		}
		h.AddGame(&moose.GameSecretMoose{
			GameName: string(body),
		})
		if _, err := fmt.Fprintf(w, "Just got this '%s' from you. '%d'\n", body, len(h.Games)); err != nil {
			log.Error(err)
		}
	})
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), nil)
	if err != nil {
		log.Fatal(err)
	}
}
