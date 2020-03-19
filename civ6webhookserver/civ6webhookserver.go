package main

import (
	"net/http"
	"os"
	"os/signal"

	"go.uber.org/zap"

	"github.com/morgabra/civ6webhook"
	"github.com/urfave/cli/v2"
)

var log, _ = zap.NewProduction()
var listenPort = "0"

func printWebhook(wh *civ6webhook.Civ6Webhook) {
	log.Info("got event",
		zap.String("game", wh.GameName),
		zap.String("player", wh.PlayerName),
		zap.String("turn", wh.TurnNumber))
}

func main() {

	app := cli.NewApp()
	app.Name = "civ6webhookserver"
	app.Version = "0.0.1"
	app.Usage = "Civilization 6 Webhook Server"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "listen-port",
			Usage:       "Webhook listen port.",
			Destination: &listenPort,
		},
	}

	app.Commands = []*cli.Command{
		Serve,
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}

var Serve = &cli.Command{
	Name:  "serve",
	Usage: "Start a webhook server and log events.",
	Action: func(ctx *cli.Context) error {
		c := civ6webhook.NewCiv6WebhookServer(log)
		defer c.Stop()

		ev, err := c.Subscribe("server")
		if err != nil {
			panic(err)
		}

		http.HandleFunc("/", c.WebhookHandler())
		go http.ListenAndServe(":"+listenPort, nil)

		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)

		for {
			select {
			case wh := <-ev:
				printWebhook(wh)
			case <-ch:
				return nil
			}
		}
	},
}
