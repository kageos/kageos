package server

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

func TestGatewayTokenCommandsRequireHRControlSignature(t *testing.T) {
	secret := strings.Repeat("g", 32)
	signer, err := controlauth.NewSigner(secret, controlauth.NATSGatewayTokenCommandScope)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := controlauth.NewVerifier(secret, controlauth.NATSGatewayTokenCommandScope, controlauth.VerifierOptions{})
	if err != nil {
		t.Fatal(err)
	}
	blacklist := &TokenBlacklist{blacklist: make(map[string]int64)}
	handler := NewTokenCommandHandler(blacklist, verifier)
	const tokenHash = "0123456789abcdef"
	payload, err := json.Marshal(InvalidateTokenMessage{
		UserID: 1, Username: "alice", Tokens: []string{tokenHash}, Timestamp: time.Now().Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}

	unsigned := nats.NewMsg(subjects.GatewayTokenInvalidateCommandSubject)
	unsigned.Data = payload
	handler.HandleTokenInvalidate(unsigned)
	if _, exists := blacklist.blacklist[tokenHash]; exists {
		t.Fatal("unsigned token command mutated the blacklist")
	}

	signed := nats.NewMsg(subjects.GatewayTokenInvalidateCommandSubject)
	signed.Data = payload
	if err := controlauth.SignNATSMessage(signed, signer); err != nil {
		t.Fatal(err)
	}
	handler.HandleTokenInvalidate(signed)
	if _, exists := blacklist.blacklist[tokenHash]; !exists {
		t.Fatal("authenticated token invalidation was not applied")
	}

	unsignedRemove := nats.NewMsg(subjects.GatewayTokenRemoveBlacklistCommandSubject)
	unsignedRemove.Data = payload
	handler.HandleRemoveBlacklist(unsignedRemove)
	if _, exists := blacklist.blacklist[tokenHash]; !exists {
		t.Fatal("unsigned remove-blacklist command restored a revoked token")
	}

	signedRemove := nats.NewMsg(subjects.GatewayTokenRemoveBlacklistCommandSubject)
	signedRemove.Data = payload
	if err := controlauth.SignNATSMessage(signedRemove, signer); err != nil {
		t.Fatal(err)
	}
	handler.HandleRemoveBlacklist(signedRemove)
	if _, exists := blacklist.blacklist[tokenHash]; exists {
		t.Fatal("authenticated remove-blacklist command was not applied")
	}
}
