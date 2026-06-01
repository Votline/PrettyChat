// Package chat user.go represent user in chat
// Stores nick, msgs, state and badges
package chat

import (
	"fmt"

	rb "prcht/internal/ringbuffer"
)

// User represents user in chat
type User struct {
	nick   string
	badges *rb.RingBuffer[string]
	msgs   *rb.RingBuffer[string]
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
			nick:   *chatmsg.Nick,
			badges: chatmsg.Status,
			msgs:   msgs,
		}
		users[*chatmsg.Nick] = user
	}

	user.msgs.Write(*chatmsg.Msg)

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

// GetBadges iterate over badges for user
func GetBadges(users map[string]*User, nick string, yield func(badge string)) error {
	const op = "chat.GetBadges"

	user, ok := users[nick]
	if !ok {
		return fmt.Errorf("%s: user not found", op)
	}

	user.badges.ForEach(yield)

	return nil
}
