package chat

import (
	"fmt"
)

func PrintMessage(users *map[string]*User, chatMsg *ChatMessage, raw []byte) error {
	const op = "chat.PrintMessage"

	if err := newMessage(*users, chatMsg, raw); err != nil {
		return fmt.Errorf("%s: error new message: %v", op, err)
	}

	fmt.Printf("%s: %s\n", *chatMsg.Nick, *chatMsg.Msg)

	return nil
}
