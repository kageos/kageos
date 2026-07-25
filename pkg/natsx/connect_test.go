package natsx

import "testing"

func TestRedactURLForLog(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: ""},
		{name: "without credentials", raw: "nats://127.0.0.1:4222", want: "nats://127.0.0.1:4222"},
		{name: "with password", raw: "nats://user:secret@nats.example:4222", want: "nats://nats.example:4222"},
		{name: "with token", raw: "nats://private-token@nats.example:4222", want: "nats://nats.example:4222"},
		{name: "with query", raw: "nats://user:secret@nats.example:4222?token=secret", want: "nats://nats.example:4222?redacted=true"},
		{name: "server list", raw: "nats://one:secret@nats-a:4222,nats://token@nats-b:4222", want: "nats://nats-a:4222,nats://nats-b:4222"},
		{name: "invalid", raw: "://bad", want: "<redacted-url>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redactURLForLog(tt.raw); got != tt.want {
				t.Fatalf("redactURLForLog() = %q, want %q", got, tt.want)
			}
		})
	}
}
