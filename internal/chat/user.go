// Package chat user.go represent user in chat
// Stores nick, msgs, state and badges
package chat

import (
	"fmt"
	"strings"

	rb "prcht/internal/ringbuffer"
)

const (
	flagSubscriber = 1 << iota
	flagModerator
	flagLeadModerator
	flagVip
)

// User represents user in chat
type User struct {
	nick     string
	marksMap uint8 // subscriber, moderator, leadModerator, vip
	msgs     *rb.RingBuffer[string]
}

// newMessage create new message for user or update existing
func newMessage(users map[string]*User, chatmsg *ChatMessage, rawMsg []byte) error {
	const op = "chat.NewMessage"

	if err := extractMessage(chatmsg, rawMsg); err != nil {
		return fmt.Errorf("%s: error extract message: %v", op, err)
	}

	user, ok := users[*chatmsg.Nick]
	if !ok {
		msgs := rb.NewRB[string](100)

		user = &User{
			nick:     *chatmsg.Nick,
			marksMap: 0,
			msgs:     msgs,
		}
		users[*chatmsg.Nick] = user
	}

	user.msgs.Write(*chatmsg.Msg)
	fillMarks(user, chatmsg)

	return nil
}

// GetMessages iterate over messages for user
func GetMessages(users map[string]*User, nick string, yield func(msg string)) error {
	const op = "chat.GetMessages"

	user, ok := users[nick]
	if !ok {
		return fmt.Errorf("%s: user not found", op)
	}

	user.msgs.ForEach(yield)

	return nil
}

func fillMarks(user *User, chatmsg *ChatMessage) {
	user.marksMap = 0
	chatmsg.Status.ForEach(func(badge string) {
		if idx := strings.IndexByte(badge, '/'); idx != -1 {
			badge = badge[:idx]
		}

		switch badge {
		case "subscriber":
			user.marksMap |= flagSubscriber
		case "moderator":
			user.marksMap |= flagModerator
		case "lead_moderator":
			user.marksMap |= flagLeadModerator
		case "vip":
			user.marksMap |= flagVip
		}
	})
}
