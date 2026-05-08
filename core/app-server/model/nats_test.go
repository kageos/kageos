package model

import "testing"

func TestNatsURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		nats Nats
		want string
	}{
		{
			name: "without auth",
			nats: Nats{Host: "127.0.0.1", Port: 4222},
			want: "nats://127.0.0.1:4222",
		},
		{
			name: "with auth",
			nats: Nats{Host: "nats.internal", Port: 4222, User: "aos", Password: "p@ss word"},
			want: "nats://aos:p%40ss%20word@nats.internal:4222",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.nats.URL(); got != tt.want {
				t.Fatalf("URL() = %q, want %q", got, tt.want)
			}
		})
	}
}
