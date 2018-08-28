package metachat

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
)

type (
	// Messenger is a common interface that must be implemented by all messenger clients.
	Messenger interface {
		Name() string
		Start() (http.Handler, error)
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

	return &Metachat{messengers: mapping, rooms: config.Rooms}, nil
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
			chats := m.getTargetChats(msg)
			for _, chat := range chats {
				err := m.messengers[chat.Messenger].Send(msg, chat.ID)
				if err != nil {
					return err
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
	for _, v := range m.messengers {
		go func(m Messenger, errChan chan error) {
			handler, err := m.Start()
			if err != nil {
				errChan <- err
			}

			if handler != nil {
				path := strings.ToLower(strings.Replace(m.Name(), " ", "-", -1))
				r.Mount(path, handler)
			}
		}(v, errChan)
	}

	go func(errChan chan error) {
		errChan <- http.ListenAndServe(":"+strconv.Itoa(m.port), r)
	}(errChan)
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
