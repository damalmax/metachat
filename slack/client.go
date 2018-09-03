package slack

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
	"github.com/pkg/errors"
	"github.com/thehadalone/metachat/metachat"
)

type (
	// Config structure.
	Config struct {
		Token             string `json:"token"`
		VerificationToken string `json:"verificationToken"`
	}

	// Client is a Slack client.
	Client struct {
		verificationToken string
		api               *slack.Client
		usersByID         userMap
		messageChan       chan metachat.Message
	}

	// TODO add read/write methods with lock encapsulation
	userMap struct {
		sync.RWMutex
		users map[string]string
	}
)

// NewClient is a Slack client constructor.
func NewClient(config Config) (*Client, error) {
	if config.Token == "" || config.VerificationToken == "" {
		return nil, errors.New("token and verification token can't be nil")
	}

	api := slack.New(config.Token)
	users, err := api.GetUsers()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	usersByID := userMap{users: make(map[string]string)}
	for _, user := range users {
		usersByID.users[user.ID] = user.RealName
	}

	return &Client{
		verificationToken: config.VerificationToken,
		api:               api,
		usersByID:         usersByID,
		messageChan:       make(chan metachat.Message, 100),
	}, nil
}

// Name returns the messenger name.
func (c *Client) Name() string {
	return "Slack"
}

// MessageChan returns a read-only message channel.
func (c *Client) MessageChan() <-chan metachat.Message {
	return c.messageChan
}

// Start starts the client main loop.
func (c *Client) Start() error {
	return nil
}

// Webhook returns HTTP handler for webhook requests.
func (c *Client) Webhook() http.Handler {
	r := chi.NewRouter()
	r.Post("/", c.handleEvents)

	return r
}

// Send sends a message to chat with the provided ID.
func (c *Client) Send(msg metachat.Message, chat string) error {
	content := convertToSlack(msg)
	_, _, err := c.api.PostMessage(chat, content, slack.PostMessageParameters{UnfurlLinks: true, Markdown: true})
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (c *Client) handleEvents(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, render.M{"error": err.Error()})
		return
	}

	event, err := slackevents.ParseEvent(json.RawMessage(body),
		slackevents.OptionVerifyToken(&slackevents.TokenComparator{VerificationToken: c.verificationToken}))

	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, render.M{"error": err.Error()})
		return
	}

	if event.Type == slackevents.URLVerification {
		var resp *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &resp)
		if err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, render.M{"error": err.Error()})
			return
		}

		render.PlainText(w, r, resp.Challenge)
		return
	}

	if event.Type == slackevents.CallbackEvent {
		if messageEvent, ok := event.InnerEvent.Data.(*slackevents.MessageEvent); ok {
			if messageEvent.Text == "" || messageEvent.User == "" {
				return
			}

			message, err := c.convertToMetachat(messageEvent)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			c.messageChan <- message
		}
	}

	// TODO handle user join

	render.JSON(w, r, render.M{})
}
