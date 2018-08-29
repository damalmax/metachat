package metachat

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
)

const chatIDCommand = "metachat chatID"

type (
	// Messenger is a common interface that must be implemented by all messenger clients.
	Messenger interface {
		Name() string
		Webhook() http.Handler
		Start() error
		MessageChan() <-chan Message
		Send(Message, string) error
	}

	// Message is a platform-independent message representation.
	Message struct {
		Messenger string
		Chat      string
		Author    string
		Text      string
	}

	// Chat represents a single messenger chat.
	Chat struct {
		Messenger string `json:"messenger"`
		ID        string `json:"id"`
	}

	// Room is a set of chats.
	Room struct {
		Name  string `json:"name"`
		Chats []Chat `json:"chats"`
	}

	// Config structure.
	Config struct {
		Port       int         `json:"port"`
		Rooms      []Room      `json:"rooms"`
		Messengers []Messenger `json:"-"`
	}

	// Metachat structure.
	Metachat struct {
		port       int
		messengers map[string]Messenger
		rooms      []Room
	}
)

// New is a Metachat constructor.
func New(config Config) (*Metachat, error) {
	if config.Port == 0 {
		return nil, errors.New("port can't be nil")
	}

	mapping := make(map[string]Messenger)
	for _, messenger := range config.Messengers {
		mapping[messenger.Name()] = messenger
	}

	return &Metachat{
		port:       config.Port,
		messengers: mapping,
		rooms:      config.Rooms,
	}, nil
}

// Start starts the Metachat main loop.
func (m *Metachat) Start() error {
	chans := make([]<-chan Message, 100)
	for _, v := range m.messengers {
		chans = append(chans, v.MessageChan())
	}

	out := merge(chans)
	errChan := make(chan error)

	m.startMessengers(errChan)

	for {
		select {
		case msg := <-out:
			if msg.isCommand() {
				err := m.handleCommand(msg)
				if err != nil {
					return err
				}
			} else {
				chats := m.getTargetChats(msg)
				for _, chat := range chats {
					err := m.messengers[chat.Messenger].Send(msg, chat.ID)
					if err != nil {
						return err
					}
				}
			}

		case err := <-errChan:
			return err
		}
	}
}

func (m *Metachat) startMessengers(errChan chan error) {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/health"))
	for _, msgr := range m.messengers {
		handler := msgr.Webhook()
		if handler != nil {
			path := "/" + strings.ToLower(strings.Replace(msgr.Name(), " ", "-", -1))
			r.Mount(path, handler)
		} else {
			go func(msgr Messenger, errChan chan error) {
				err := msgr.Start()
				if err != nil {
					errChan <- err
				}
			}(msgr, errChan)
		}
	}

	go func(errChan chan error) {
		errChan <- http.ListenAndServe(":"+strconv.Itoa(m.port), r)
	}(errChan)
}

func (m *Metachat) handleCommand(msg Message) error {
	if msg.Text == chatIDCommand {
		err := m.messengers[msg.Messenger].Send(Message{Text: msg.Chat}, msg.Chat)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Metachat) getTargetChats(msg Message) []Chat {
	result := make([]Chat, 0)
	for _, room := range m.rooms {
		if !isMessageFromRoom(msg, room) {
			continue
		}

		for _, chat := range room.Chats {
			if chat.Messenger != msg.Messenger || chat.ID != msg.Chat {
				result = append(result, chat)
			}
		}
	}

	return result
}

func (m Message) isCommand() bool {
	return m.Text == chatIDCommand
}

func isMessageFromRoom(msg Message, room Room) bool {
	for _, chat := range room.Chats {
		if msg.Chat == chat.ID {
			return true
		}
	}

	return false
}

func merge(chans []<-chan Message) <-chan Message {
	out := make(chan Message)

	for _, c := range chans {
		go func(c <-chan Message) {
			for v := range c {
				out <- v
			}
		}(c)
	}

	return out
}
