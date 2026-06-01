// Package chat msg.go extract nick, msg, state and badges from message
package chat

import (
	"bytes"
	"fmt"
	"unsafe"

	rb "prcht/internal/ringbuffer"
)

// ChatMessage contains data for one message
// nick, msg and state are pointers to avoid allocations
// Join contains channel name, e.g. '#bratishkinoff :'
type ChatMessage struct {
	Nick   *string
	Msg    *string
	Join   []byte
	Status *rb.RingBuffer[string]
}

// ExtractMessage extract nick and msg from raw message
// Reuse buffers to avoid allocations.
func extractMessage(chat *ChatMessage, rawMsg []byte) error {
	const op = "chat.TrimNick"

	if !bytes.Contains(rawMsg, []byte("PRIVMSG")) {
		return fmt.Errorf("%s: not PRIVMSG message", op)
	}

	if err := extractNick(rawMsg, chat.Nick); err != nil {
		return fmt.Errorf("%s: error extract nick: %v", op, err)
	}

	if err := extractMsg(rawMsg, chat.Join, chat.Msg); err != nil {
		return fmt.Errorf("%s: error extract msg: %v", op, err)
	}

	if err := extractBadges(rawMsg, chat.Status); err != nil {
		return fmt.Errorf("%s: error extract badges: %v", op, err)
	}

	return nil
}

// extractNick find nick in raw message by 'user-type=' pattern
func extractNick(rawMsg []byte, nick *string) error {
	const op = "chat.extractNick"

	contentStart := bytes.Index(rawMsg, []byte("user-type="))
	if contentStart == -1 {
		return fmt.Errorf("%s: user-type not found", op)
	}
	contentStart += len("user-type=")
	nameStart := bytes.IndexByte(rawMsg[contentStart:], ':')
	if nameStart == -1 {
		return fmt.Errorf("%s: name not found", op)
	}
	nameStart += contentStart + 1 // +1 for skip ':'
	nameEnd := bytes.IndexByte(rawMsg[nameStart:], '!')
	if nameEnd == -1 {
		return fmt.Errorf("%s: name not found", op)
	}
	nameEnd += nameStart

	nickBytes := rawMsg[nameStart:nameEnd]
	*nick = unsafe.String(unsafe.SliceData(nickBytes), len(nickBytes))

	return nil
}

// extractMsg find msg in raw message by 'PRIVMSG' and 'join' pattern
// 'join' contains channel name, e.g. '#bratishkinoff :'
func extractMsg(rawMsg, join []byte, msg *string) error {
	const op = "chat.extractMsg"

	contentStart := bytes.Index(rawMsg, []byte("PRIVMSG"))
	if contentStart == -1 {
		return fmt.Errorf("%s: PRIVMSG not found", op)
	}
	contentStart += len("PRIVMSG")

	msgStart := bytes.Index(rawMsg[contentStart:], join)
	if msgStart == -1 {
		return fmt.Errorf("%s: join not found", op)
	}
	msgStart += contentStart + len(join)
	msgEnd := len(rawMsg)

	msgBytes := rawMsg[msgStart:msgEnd]
	*msg = unsafe.String(unsafe.SliceData(msgBytes), len(msgBytes))

	return nil
}

// extractBadges find badges by 'badges=' pattern
// and set state to subscriber, leadMod or moderator
func extractBadges(rawMsg []byte, badges *rb.RingBuffer[string]) error {
	const op = "chat.extractBadges"

	start := bytes.Index(rawMsg, []byte("badges="))
	if start == -1 {
		return fmt.Errorf("%s: badges not found", op)
	}
	start += len("badges=")

	end := bytes.IndexByte(rawMsg[start:], ';')
	if end == -1 {
		return fmt.Errorf("%s: badges not found", op)
	}
	end += start

	badgesContent := rawMsg[start:end]

	rangeByByte(badgesContent, ',', func(badge []byte) {
		badgeStr := unsafe.String(unsafe.SliceData(badge), len(badge))
		badges.Write(badgeStr)
	})

	return nil
}

// rangeByByte iterates over content by byte and calls yield for each splice
func rangeByByte(content []byte, sep byte, yield func(splice []byte)) {
	for len(content) > 0 {
		idx := bytes.IndexByte(content, sep)
		if idx == -1 {
			yield(content)
			return
		}
		yield(content[:idx])
		content = content[idx+1:]
	}
}
