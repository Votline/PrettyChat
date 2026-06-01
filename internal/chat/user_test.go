package chat

import (
	"testing"

	rb "prcht/internal/ringbuffer"
)

func TestFillMarks(t *testing.T) {
	const op = "chat.TestFillMarks"

	tests := []struct {
		name string
		in   string
		want uint8
	}{
		{
			name: "subscriber",
			in:   "subscriber/12",
			want: flagSubscriber,
		},
		{
			name: "moderator",
			in:   "moderator/6",
			want: flagModerator,
		},
		{
			name: "lead_moderator",
			in:   "lead_moderator/18",
			want: flagLeadModerator,
		},
		{
			name: "vip",
			in:   "vip",
			want: flagVip,
		},
		{
			name: "nop",
			in:   "",
			want: 0,
		},
	}

	for _, tt := range tests {
		user := User{
			nick: "test",
		}
		chatmsg := ChatMessage{
			Status: rb.NewRB[string](10),
		}

		chatmsg.Status.Write(tt.in)
		fillMarks(&user, &chatmsg)

		if user.marksMap != tt.want {
			t.Errorf("%s: fillMarks = %d, want %d", tt.name, user.marksMap, tt.want)
		}
	}
}

func BenchmarkFillMarks(b *testing.B) {
	const op = "chat.BenchmarkFillMarks"

	user := User{
		nick: "test",
	}
	chatmsg := ChatMessage{
		Status: rb.NewRB[string](10),
	}
	chatmsg.Status.Write("subscriber/12")
	chatmsg.Status.Write("moderator/6")
	chatmsg.Status.Write("lead_moderator/18")
	chatmsg.Status.Write("vip")

	b.ResetTimer()
	for b.Loop() {
		fillMarks(&user, &chatmsg)
	}
}
