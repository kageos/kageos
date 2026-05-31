package app

import (
	"testing"

	"github.com/kageos/kageos/pkg/trace"
)

func TestShouldBindGETPrefersQueryOverBody(t *testing.T) {
	ctx := &Context{
		msg:      &trace.Msg{Method: "GET"},
		body:     []byte(`{}`),
		urlQuery: "owner=octocat&repo=Hello-World",
	}
	var req struct {
		Owner string `form:"owner" json:"owner"`
		Repo  string `form:"repo" json:"repo"`
	}

	if err := ctx.ShouldBind(&req); err != nil {
		t.Fatalf("ShouldBind() error = %v, want nil", err)
	}
	if req.Owner != "octocat" || req.Repo != "Hello-World" {
		t.Fatalf("req = %+v, want query values", req)
	}
}

func TestShouldBindPOSTAllowsEmptyBody(t *testing.T) {
	ctx := &Context{msg: &trace.Msg{Method: "POST"}}
	var req struct{}

	if err := ctx.ShouldBind(&req); err != nil {
		t.Fatalf("ShouldBind() error = %v, want nil", err)
	}
}
