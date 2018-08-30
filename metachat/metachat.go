package metachat

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
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
		rooms      map[string]Room
	}
)

// New is a Metachat constructor.
func New(config Config) (*Metachat, error) {
	if config.Port == 0 {
		return nil, errors.New("port can't be nil")
	}

	messengers := make(map[string]Messenger)
	for _, messenger := range config.Messengers {
		messengers[niceName(messenger.Name())] = messenger
	}

	rooms := make(map[string]Room)
	for _, room := range config.Rooms {
		rooms[niceName(room.Name)] = room
	}

	metachat := &Metachat{
		port:       config.Port,
		messengers: messengers,
		rooms:      rooms,
	}

	if err := metachat.validate(); err != nil {
		return nil, err
	}

	return metachat, nil
}

// Start starts the Metachat main loop.
func (m *Metachat) Start() error {
	chans := make([]<-chan Message, 0)
	for _, v := range m.messengers {
		chans = append(chans, v.MessageChan())
	}

	out := merge(chans)
	errChan := make(chan error)

	m.registerHandlers(errChan)
	m.startMessengers(errChan)

	for {
		select {
		case msg := <-out:
			if isCommand(msg) {
				err := m.handleCommand(msg)
				if err != nil {
					return err
				}
			} else {
				chats := m.getTargetChats(msg)
				for _, chat := range chats {
					err := m.messengers[niceName(chat.Messenger)].Send(msg, chat.ID)
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

func (m *Metachat) validate() error {
	for _, room := range m.rooms {
		for _, chat := range room.Chats {
			if _, ok := m.messengers[niceName(chat.Messenger)]; !ok {
				return errors.Errorf("messenger '%s' from room '%s' not found", chat.Messenger, room.Name)
			}
		}
	}

	return nil
}

func (m *Metachat) registerHandlers(errChan chan error) {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/health"))
	r.Post("/rooms/{room}", m.postMessageHandler)

	for _, msgr := range m.messengers {
		handler := msgr.Webhook()
		if handler != nil {
			path := "/" + niceName(msgr.Name())
			r.Mount(path, handler)
		}
	}

	go func(errChan chan error) {
		errChan <- http.ListenAndServe(":"+strconv.Itoa(m.port), r)
	}(errChan)
}

func (m *Metachat) startMessengers(errChan chan error) {
	for _, msgr := range m.messengers {
		go func(msgr Messenger, errChan chan error) {
			err := msgr.Start()
			if err != nil {
				errChan <- err
			}
		}(msgr, errChan)
	}
}

func (m *Metachat) handleCommand(msg Message) error {
	if msg.Text == chatIDCommand {
		err := m.messengers[niceName(msg.Messenger)].Send(Message{Text: msg.Chat}, msg.Chat)
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

func (m *Metachat) postMessageHandler(w http.ResponseWriter, r *http.Request) {
	roomName := chi.URLParam(r, "room")
	room, ok := m.rooms[roomName]
	if !ok {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, render.M{})
		return
	}

	message := Message{}
	if err := render.Decode(r, &message); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, render.M{"error": err.Error()})
		return
	}

	message.Author = ""

	for _, chat := range room.Chats {
		err := m.messengers[niceName(chat.Messenger)].Send(message, chat.ID)
		if err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, render.M{"error": err.Error()})
			return
		}
	}

	render.JSON(w, r, render.M{})
}

func isCommand(message Message) bool {
	return message.Text == chatIDCommand
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

func niceName(name string) string {
	return strings.ToLower(strings.Replace(name, " ", "-", -1))
}
