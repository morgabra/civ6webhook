package civ6webhook

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// Civ6Webhook is the game state webhook fired by the 'cloud' multiplayer mode
// { "value1":"game name", "value2":"player name", "value3":"game turn number" }
type Civ6Webhook struct {
	GameName   string `json:"value1"`
	PlayerName string `json:"value2"`
	TurnNumber string `json:"value3"`
}

type Civ6WebhookServer struct {
	log *zap.Logger

	subs map[string]chan<- *Civ6Webhook

	mtx       sync.Mutex
	closeOnce sync.Once
}

func (c *Civ6WebhookServer) WebhookHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			c.log.Error("cannot read request body", zap.Error(err))
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		c.log.Info("got webhook request", zap.ByteString("body", body))

		wh := Civ6Webhook{}

		err = json.Unmarshal(body, &wh)
		if err != nil {
			c.log.Error("error parsing body", zap.Error(err))
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		for name, ch := range c.subs {
			whCopy := wh
			select {
			case ch <- &whCopy:
			default:
				c.log.Error("failed sending notification to subscriber", zap.String("subscriber", name))
			}
		}

	}
}

func (c *Civ6WebhookServer) Subscribe(name string) (<-chan *Civ6Webhook, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	name = strings.ToLower(name)

	_, ok := c.subs[name]
	if ok {
		return nil, errors.New("name is already subscribed")
	}

	ch := make(chan *Civ6Webhook, 5)
	c.subs[name] = ch
	return ch, nil
}

func (c *Civ6WebhookServer) Stop() {

	c.closeOnce.Do(func() {
		for _, ch := range c.subs {
			close(ch)
		}

		c.log.Sync()
	})

}

func NewCiv6WebhookServer(log *zap.Logger) *Civ6WebhookServer {

	if log == nil {
		log, _ = zap.NewProduction()
	}

	s := &Civ6WebhookServer{
		log:  log,
		subs: map[string]chan<- *Civ6Webhook{},
	}

	return s
}
