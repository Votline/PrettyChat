package chat

import (
	"fmt"
)

func PrintMessage(users *map[string]*User, chatMsg *ChatMessage, raw []byte) error {
	const op = "chat.PrintMessage"

	if err := newMessage(*users, chatMsg, raw); err != nil {
		return fmt.Errorf("%s: error new message: %v", op, err)
	}

	if err := prettyPrint(users, chatMsg, raw); err != nil {
		return fmt.Errorf("%s: error pretty print: %v", op, err)
	}

	return nil
}

func prettyPrint(users *map[string]*User, chatMsg *ChatMessage, raw []byte) error {
	const op = "chat.PrettyPrint"

	user, ok := (*users)[*chatMsg.Nick]
	if !ok {
		return fmt.Errorf("%s: user not found", op)
	}

	markStr := ""
	if user.marksMap&flagLeadModerator != 0 {
		markStr += "LM"
	}
	if user.marksMap&flagModerator != 0 {
		markStr += "M"
	}
	if user.marksMap&flagSubscriber != 0 {
		markStr += "S"
	}
	if user.marksMap&flagVip != 0 {
		markStr += "V"
	}
	if len(markStr) > 0 {
		markStr += " "
	}

	fmt.Printf("%s%s %s\n", markStr, *chatMsg.Nick, *chatMsg.Msg)

	return nil
}
