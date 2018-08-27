package slack

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
	"github.com/pkg/errors"
	"github.com/thehadalone/metachat/metachat"
)

type (
	// Config structure.
	Config struct {
		Port              int    `json:"port"`
		Token             string `json:"token"`
		VerificationToken string `json:"verificationToken"`
	}

	// Client is a Slack HTTP client.
	Client struct {
		port              int
		verificationToken string
		api               *slack.Client
		usersByID         userMap
		messageChan       chan metachat.Message
	}

	userMap struct {
		sync.RWMutex
		users map[string]string
	}
)

// NewClient is a Slack client constructor.
func NewClient(config Config) (*Client, error) {
	if config.Token == "" || config.VerificationToken == "" || config.Port == 0 {
		return nil, errors.New("token, verification token and port can't be nil")
	}

	api := slack.New(config.Token)
	users, err := api.GetUsers()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	usersByID := userMap{users: make(map[string]string)}
	usersByID.Lock()
	for _, user := range users {
		usersByID.users[user.ID] = user.RealName
	}
	usersByID.Unlock()

	return &Client{
		port:              config.Port,
		verificationToken: config.VerificationToken,
		api:               api,
		usersByID:         usersByID,
		messageChan:       make(chan metachat.Message, 100),
	}, nil
}

// Type returns the messenger type.
func (c *Client) Type() string {
	return "Slack"
}

// MessageChan returns a read-only message channel.
func (c *Client) MessageChan() <-chan metachat.Message {
	return c.messageChan
}

// Start starts the client main loop.
func (c *Client) Start() error {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	r.Post("/", c.handleEvents)

	return http.ListenAndServe(":"+strconv.Itoa(c.port), r)
}

// Send sends a message to chat with the provided ID.
func (c *Client) Send(message metachat.Message, chat string) error {
	_, _, err := c.api.PostMessage(chat, fmt.Sprintf("*[%s]* %s", message.Author, message.Text),
		slack.PostMessageParameters{UnfurlLinks: true, Markdown: true})

	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (c *Client) handleEvents(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	event, err := slackevents.ParseEvent(json.RawMessage(body),
		slackevents.OptionVerifyToken(&slackevents.TokenComparator{VerificationToken: c.verificationToken}))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if event.Type == slackevents.URLVerification {
		var resp *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		render.PlainText(w, r, resp.Challenge)
		return
	}

	if event.Type == slackevents.CallbackEvent {
		if messageEvent, ok := event.InnerEvent.Data.(*slackevents.MessageEvent); ok {
			message, err := c.convert(messageEvent)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			c.messageChan <- message
			return
		}
	}
}

func (c *Client) convert(event *slackevents.MessageEvent) (metachat.Message, error) {
	c.usersByID.RLock()
	author, ok := c.usersByID.users[event.User]
	c.usersByID.RUnlock()
	if !ok {
		user, err := c.api.GetUserInfo(event.User)
		if err != nil {
			return metachat.Message{}, errors.WithStack(err)
		}

		author = user.RealName
		c.usersByID.Lock()
		c.usersByID.users[user.ID] = user.RealName
		c.usersByID.Unlock()
	}

	return metachat.Message{
		Messenger: "Slack",
		Chat:      event.Channel,
		Author:    author,
		Text:      event.Text,
	}, nil
}
