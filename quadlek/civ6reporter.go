package civ6reporter

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/jirwin/quadlek/quadlek"
	"github.com/morgabra/civ6webhook"
)

var civ6WebhookServer *civ6webhook.Civ6WebhookServer
var civ6WebhookHandler http.HandlerFunc

func help(cmdMsg *quadlek.CommandMsg) {
	cmdMsg.Command.Reply() <- &quadlek.CommandResp{
		Text:      "civ6reporter: report civ 6 cloud-play games.\nAvailable commands: help, last",
		InChannel: false,
	}
}

func sayError(cmdMsg *quadlek.CommandMsg, msg string, inChannel bool) {
	cmdMsg.Command.Reply() <- &quadlek.CommandResp{
		Text:      fmt.Sprintf("Uh Oh. Something broke: %s", msg),
		InChannel: inChannel,
	}
}

func say(cmdMsg *quadlek.CommandMsg, msg string, inChannel bool) {
	cmdMsg.Command.Reply() <- &quadlek.CommandResp{
		Text:      msg,
		InChannel: inChannel,
	}
}

func civ6ReporterWebhook(ctx context.Context, whChannel <-chan *quadlek.WebhookMsg) {
	for {
		select {
		case whMsg := <-whChannel:
			civ6WebhookHandler(whMsg.ResponseWriter, whMsg.Request)
			whMsg.Done <- true
		case <-ctx.Done():
			log.Info("civ6reporter: stopping webhook handler")
			if civ6WebhookServer != nil {
				civ6WebhookServer.Stop()
			}
			return
		}
	}
}

func civ6ReporterCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:

			// /twitch <command> <args...>
			cmd := strings.SplitN(cmdMsg.Command.Text, " ", 1)
			if len(cmd) == 0 {
				help(cmdMsg)
				return
			}
			log.Infof("civ6reporter: got command %s", cmd[0])
			switch cmd[0] {
			case "last":
				cmdMsg.Command.Reply() <- &quadlek.CommandResp{
					Text:      "NotImplemented",
					InChannel: true,
				}
			default:
				help(cmdMsg)
			}

		case <-ctx.Done():
			log.Info("civ6reporter: stopping command handler")
			if civ6WebhookServer != nil {
				civ6WebhookServer.Stop()
			}
			return
		}
	}
}

func watch(bot *quadlek.Bot, channels []string, userMap map[string]string, ch <-chan *civ6webhook.Civ6Webhook) {
	for {
		select {
		case wh, ok := <-ch:
			if !ok {
				return
			}
			for _, scn := range channels {
				scid, err := bot.GetChannelId(scn)
				if err != nil {
					log.WithError(err).Errorf("twitch: got stream event, but failed looking up slack channel id %s", scn)
					continue
				}

				slackUser, ok := userMap[strings.ToLower(wh.PlayerName)]
				if ok {
					wh.PlayerName = slackUser
				}

				bot.Say(scid, fmt.Sprintf("civ6reporter: hey %s, it's your turn! (game: %s, turn: %s)", wh.PlayerName, wh.GameName, wh.TurnNumber))
			}
		}
	}
}

func load(channels []string, userMap map[string]string) func(bot *quadlek.Bot, store *quadlek.Store) error {

	return func(bot *quadlek.Bot, store *quadlek.Store) error {
		// TODO: store whos turn it for the 'last' command
		ev, err := civ6WebhookServer.Subscribe("civ6reporter")
		if err != nil {
			return err
		}

		// munge the user map
		for k, v := range userMap {
			userMap[strings.ToLower(k)] = v
		}

		go watch(bot, channels, userMap, ev)

		return nil
	}
}

func Register(channels []string, userMap map[string]string) quadlek.Plugin {

	civ6WebhookServer = civ6webhook.NewCiv6WebhookServer(nil)
	civ6WebhookHandler = civ6WebhookServer.WebhookHandler()

	return quadlek.MakePlugin(
		"civ6reporter",
		[]quadlek.Command{
			quadlek.MakeCommand("civ6reporter", civ6ReporterCommand),
		},
		nil,
		nil,
		[]quadlek.Webhook{
			quadlek.MakeWebhook("civ6reporter", civ6ReporterWebhook),
		},
		load(channels, userMap),
	)
}
