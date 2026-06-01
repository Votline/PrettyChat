package chat

import (
	"testing"

	rb "prcht/internal/ringbuffer"
)

const msg = `
@badge-info=;badges=subscriber;client-nonce=42004b38-c5ae-4d0b-bb9c-fc92ff186b9a;color=;display-name=test;emotes=;first-msg=0;flags=;id=704cbab3-1db4-479f-b0ac-22c80e9fa405;mod=0;returning-chatter=0;room-id=120304051;subscriber=0;tmi-sent-ts=1780241123521;turbo=0;user-id=1289147469;user-type= :test!test@test.tmi.twitch.tv PRIVMSG #nop :test msg
`

func TestExtractNick(t *testing.T) {
	const op = "chat.TestExtractNick"

	tests := []struct {
		testName string
		rawMsg   []byte
		nick     *string
		wantNick string
		wantErr  bool
	}{
		{
			testName: "full msg",
			rawMsg:   []byte(msg),
			nick:     new(string),
			wantNick: "test",
			wantErr:  false,
		},
		{
			testName: "only PRIVMSG",
			rawMsg:   []byte("PRIVMSG"),
			nick:     new(string),
			wantNick: "",
			wantErr:  true,
		},
		{
			testName: "no badges",
			rawMsg:   []byte("user-type= :test!test@test.tmi.twitch.tv PRIVMSG #nop :test msg"),
			nick:     new(string),
			wantNick: "test",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.wantNick, func(t *testing.T) {
			if err := extractNick(tt.rawMsg, tt.nick); (err != nil) != tt.wantErr {
				t.Errorf("%s: extractNick() error = %v, wantErr %v", tt.testName, err, tt.wantErr)
			}
			if *tt.nick != tt.wantNick {
				t.Errorf("%s: extractNick() got = %q, want %q", tt.testName, *tt.nick, tt.wantNick)
			}
		})
	}
}

func BenchmarkExtractNick(b *testing.B) {
	const op = "chat.BenchmarkExtractNick"

	msgBytes := []byte(msg)

	b.ResetTimer()
	for b.Loop() {
		if err := extractNick(msgBytes, new(string)); err != nil {
			b.Errorf("%s: extractNick() error = %v", op, err)
		}
	}
}

func TestExtractMsg(t *testing.T) {
	const op = "chat.TestExtractMsg"

	tests := []struct {
		testName string
		rawMsg   []byte
		join     []byte
		msg      *string
		wantMsg  string
		wantErr  bool
	}{
		{
			testName: "full msg",
			rawMsg:   []byte(msg),
			join:     []byte("#nop :"),
			msg:      new(string),
			wantMsg:  "test msg\n",
			wantErr:  false,
		},
		{
			testName: "only PRIVMSG",
			rawMsg:   []byte("PRIVMSG"),
			join:     []byte("#nop :"),
			msg:      new(string),
			wantMsg:  "",
			wantErr:  true,
		},
		{
			testName: "no user info",
			rawMsg:   []byte("PRIVMSG #nop :test msg"),
			join:     []byte("#nop :"),
			msg:      new(string),
			wantMsg:  "test msg",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		if err := extractMsg(tt.rawMsg, tt.join, tt.msg); (err != nil) != tt.wantErr {
			t.Errorf("%s: extractMsg() error = %v, wantErr %v", tt.testName, err, tt.wantErr)
		}
		if *tt.msg != tt.wantMsg {
			t.Errorf("%s: extractMsg() got = %q, want %q", tt.testName, *tt.msg, tt.wantMsg)
		}
	}
}

func BenchmarkExtractMsg(b *testing.B) {
	const op = "chat.BenchmarkExtractMsg"

	msgBytes := []byte(msg)
	joinBytes := []byte("#nop :")

	b.ResetTimer()
	for b.Loop() {
		if err := extractMsg(msgBytes, joinBytes, new(string)); err != nil {
			b.Errorf("%s: extractMsg() error = %v", op, err)
		}
	}
}

func TestExtractBadges(t *testing.T) {
	const op = "chat.TestExtractBadges"

	tests := []struct {
		testName   string
		rawMsg     []byte
		badges     *rb.RingBuffer[string]
		wantBadges [10]string
		wantErr    bool
	}{
		{
			testName: "full msg",
			rawMsg:   []byte(msg),
			badges:   rb.NewRB[string](10),
			wantBadges: [10]string{
				"subscriber",
			},
			wantErr: false,
		},
		{
			testName:   "only PRIVMSG",
			rawMsg:     []byte("PRIVMSG"),
			badges:     rb.NewRB[string](10),
			wantBadges: [10]string{},
			wantErr:    true,
		},
		{
			testName:   "no badges",
			rawMsg:     []byte("user-type= :test!test@test.tmi.twitch.tv PRIVMSG #nop :test msg"),
			badges:     rb.NewRB[string](10),
			wantBadges: [10]string{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		if err := extractBadges(tt.rawMsg, tt.badges); (err != nil) != tt.wantErr {
			t.Errorf("%s: extractBadges error = %v, wantErr %v", tt.testName, err, tt.wantErr)
			continue
		}
		var val string
		tt.badges.Read(&val)
		if val != tt.wantBadges[0] {
			t.Errorf("%s: extractBadges got len = %q, want %q", tt.testName, val, tt.wantBadges)
			continue
		}
	}
}

func BenchmarkExtractBadges(b *testing.B) {
	const op = "chat.BenchmarkExtractBadges"

	msgBytes := []byte(msg)

	b.ResetTimer()
	for b.Loop() {
		if err := extractBadges(msgBytes, rb.NewRB[string](10)); err != nil {
			b.Errorf("%s: extractBadges() error = %v", op, err)
		}
	}
}
