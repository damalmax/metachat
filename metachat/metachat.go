package metachat

type (
	// Messenger is a common interface that must be implemented by all messenger clients.
	Messenger interface {
		Type() string
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
		Rooms []Room `json:"rooms"`
	}

	// Metachat structure.
	Metachat struct {
		messengers map[string]Messenger
		rooms      []Room
	}
)

// New is a metachat constructor.
func New(rooms []Room, messengers ...Messenger) *Metachat {
	mapping := make(map[string]Messenger)
	for _, messenger := range messengers {
		mapping[messenger.Type()] = messenger
	}

	return &Metachat{messengers: mapping, rooms: rooms}
}

// Start starts the metachat main loop.
func (m *Metachat) Start() error {
	chans := make([]<-chan Message, 100)
	for _, v := range m.messengers {
		chans = append(chans, v.MessageChan())
	}

	out := merge(chans)
	errChan := make(chan error)

	for _, v := range m.messengers {
		go func(m Messenger, errChan chan error) {
			err := m.Start()
			if err != nil {
				errChan <- err
			}
		}(v, errChan)
	}

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
