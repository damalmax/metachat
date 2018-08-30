package telegram

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/render"

	"github.com/go-chi/chi"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
	"github.com/thehadalone/metachat/metachat"
)

type (
	// Config structure.
	Config struct {
		Token string `json:"token"`
	}

	// Client is a Telegram client.
	Client struct {
		api         *tgbotapi.BotAPI
		messageChan chan metachat.Message
	}
)

// NewClient is a Telegram client constructor.
func NewClient(config Config) (*Client, error) {
	if config.Token == "" {
		return nil, errors.New("token can't be nil")
	}

	api, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &Client{api: api, messageChan: make(chan metachat.Message, 100)}, nil
}

// Name returns the messenger name.
func (c *Client) Name() string {
	return "Telegram"
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

// MessageChan returns a read-only message channel.
func (c *Client) MessageChan() <-chan metachat.Message {
	return c.messageChan
}

// Send sends a message to chat with the provided ID.
func (c *Client) Send(message metachat.Message, chat string) error {
	id, err := strconv.ParseInt(chat, 10, 64)
	if err != nil {
		return errors.WithStack(err)
	}

	content := message.Text
	if message.Author != "" {
		content = fmt.Sprintf("*[%s]* %s", message.Author, message.Text)
	}

	msg := tgbotapi.NewMessage(id, content)
	msg.ParseMode = tgbotapi.ModeMarkdown

	_, err = c.api.Send(msg)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (c *Client) handleEvents(w http.ResponseWriter, r *http.Request) {
	var event tgbotapi.Update
	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, render.M{"error": err.Error()})
		return
	}

	if event.Message == nil || event.Message.Text == "" {
		return
	}

	c.messageChan <- convert(event.Message)
	render.JSON(w, r, render.M{})
}

func convert(msg *tgbotapi.Message) metachat.Message {
	return metachat.Message{
		Messenger: "Telegram",
		Chat:      strconv.FormatInt(msg.Chat.ID, 10),
		Author:    fmt.Sprintf("%s %s", msg.From.FirstName, msg.From.LastName),
		Text:      msg.Text,
	}
}
