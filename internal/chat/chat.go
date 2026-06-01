package chat

import (
	"fmt"
	"strings"
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

	mark := make(map[string]struct{})

	if err := GetBadges(*users, *chatMsg.Nick, func(badge string) {
		if idx := strings.IndexByte(badge, '/'); idx != -1 {
			badge = badge[:idx]
		}

		switch badge {
		case "subscriber":
			mark["S"] = struct{}{}
		case "moderator":
			mark["M"] = struct{}{}
		case "lead_moderator":
			mark["LM"] = struct{}{}
		case "vip":
			mark["V"] = struct{}{}
		}
	}); err != nil {
		return fmt.Errorf("%s: error get badges: %v", op, err)
	}

	markStr := ""
	for k := range mark {
		markStr += k + " "
	}

	fmt.Printf("%s%s %s\n", markStr, *chatMsg.Nick, *chatMsg.Msg)

	return nil
}
