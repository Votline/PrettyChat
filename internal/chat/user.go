// Package chat user.go represent user in chat
// Stores nick, msgs, state and badges
package chat

import (
	"fmt"
)

// User represents user in chat
type User struct {
	Nick   string
	Badges [10]string
	msgs   [100]string
}

// newMessage create new message for user or update existing
func newMessage(users map[string]*User, chatmsg *ChatMessage, rawMsg []byte) error {
	const op = "chat.NewMessage"

	if err := extractMessage(chatmsg, rawMsg); err != nil {
		return fmt.Errorf("%s: error extract message: %v", op, err)
	}

	user, ok := users[*chatmsg.Nick]
	if !ok {
		msgs := [100]string{}

		user = &User{
			Nick:   *chatmsg.Nick,
			Badges: *chatmsg.Status,
			msgs:   msgs,
		}
		users[*chatmsg.Nick] = user
	}

	user.msgs[len(user.msgs)-1] = *chatmsg.Msg

	return nil
}

// getMessages iterate over messages for user
func getMessages(users map[string]*User, nick string, yield func(msg string)) error {
	const op = "chat.GetMessages"

	user, ok := users[nick]
	if !ok {
		return fmt.Errorf("%s: user not found", op)
	}

	for _, msg := range user.msgs {
		yield(msg)
	}

	return nil
}

// getBadges iterate over badges for user
func getBadges(users map[string]*User, nick string, yield func(badge string)) error {
	const op = "chat.GetBadges"

	user, ok := users[nick]
	if !ok {
		return fmt.Errorf("%s: user not found", op)
	}

	for _, badge := range user.Badges {
		yield(badge)
	}

	return nil
}
