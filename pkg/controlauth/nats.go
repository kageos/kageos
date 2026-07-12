package controlauth

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net/textproto"
	"sort"
	"strconv"
	"strings"

	"github.com/nats-io/nats.go"
)

const (
	// NATSGatewayTokenCommandScope authenticates HR-server commands that mutate
	// the API gateway's access-token blacklist.
	NATSGatewayTokenCommandScope = "hr-server/gateway-token-command-nats/v1"

	NATSVersionHeader   = "X-Kageos-Control-Version"
	NATSTimestampHeader = "X-Kageos-Control-Timestamp"
	NATSNonceHeader     = "X-Kageos-Control-Nonce"
	NATSSignatureHeader = "X-Kageos-Control-Signature"
)

var natsAuthHeaders = map[string]struct{}{
	strings.ToLower(NATSVersionHeader):   {},
	strings.ToLower(NATSTimestampHeader): {},
	strings.ToLower(NATSNonceHeader):     {},
	strings.ToLower(NATSSignatureHeader): {},
}

// SignNATSMessage protects the subject, reply inbox, payload, timestamp, nonce,
// and all non-authentication headers. Request callers must assign Reply before
// signing so a shared-bus listener cannot redirect an authenticated command's
// response to an attacker-controlled inbox.
func SignNATSMessage(msg *nats.Msg, signer *Signer) error {
	if msg == nil {
		return fmt.Errorf("controlauth: nats message is nil")
	}
	if signer == nil {
		return ErrInvalidConfig
	}
	if msg.Header == nil {
		msg.Header = nats.Header{}
	}
	deleteNATSAuthHeaders(msg.Header)
	canonicalHeaders, err := canonicalNATSHeaders(msg.Header)
	if err != nil {
		return err
	}
	metadata, err := signer.Sign(
		[]byte(msg.Subject),
		[]byte(msg.Reply),
		msg.Data,
		canonicalHeaders,
	)
	if err != nil {
		return err
	}
	msg.Header.Set(NATSVersionHeader, metadata.Version)
	msg.Header.Set(NATSTimestampHeader, strconv.FormatInt(metadata.Timestamp, 10))
	msg.Header.Set(NATSNonceHeader, metadata.Nonce)
	msg.Header.Set(NATSSignatureHeader, metadata.Signature)
	return nil
}

// VerifyNATSMessage validates and consumes the message nonce. Verification is
// fail-closed: unsigned, stale, tampered, or replayed messages return an error.
func VerifyNATSMessage(msg *nats.Msg, verifier *Verifier) error {
	if msg == nil {
		return fmt.Errorf("controlauth: nats message is nil")
	}
	if verifier == nil {
		return ErrInvalidConfig
	}
	metadata, err := natsMetadata(msg.Header)
	if err != nil {
		return err
	}
	canonicalHeaders, err := canonicalNATSHeaders(msg.Header)
	if err != nil {
		return err
	}
	return verifier.Verify(
		metadata,
		[]byte(msg.Subject),
		[]byte(msg.Reply),
		msg.Data,
		canonicalHeaders,
	)
}

func natsMetadata(header nats.Header) (Metadata, error) {
	version, err := singleNATSHeader(header, NATSVersionHeader)
	if err != nil {
		return Metadata{}, err
	}
	timestampText, err := singleNATSHeader(header, NATSTimestampHeader)
	if err != nil {
		return Metadata{}, err
	}
	nonce, err := singleNATSHeader(header, NATSNonceHeader)
	if err != nil {
		return Metadata{}, err
	}
	signature, err := singleNATSHeader(header, NATSSignatureHeader)
	if err != nil {
		return Metadata{}, err
	}
	timestamp, err := strconv.ParseInt(timestampText, 10, 64)
	if err != nil {
		return Metadata{}, ErrInvalidTimestamp
	}
	return Metadata{Version: version, Timestamp: timestamp, Nonce: nonce, Signature: signature}, nil
}

func singleNATSHeader(header nats.Header, name string) (string, error) {
	if len(header) == 0 {
		return "", ErrMissingMetadata
	}
	values := header.Values(name)
	if len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return "", ErrMissingMetadata
	}
	return values[0], nil
}

func deleteNATSAuthHeaders(header nats.Header) {
	for name := range header {
		if _, ok := natsAuthHeaders[strings.ToLower(name)]; ok {
			delete(header, name)
		}
	}
}

func canonicalNATSHeaders(header nats.Header) ([]byte, error) {
	type entry struct {
		name   string
		values []string
	}
	entries := make([]entry, 0, len(header))
	seenNames := make(map[string]string, len(header))
	for name, values := range header {
		canonicalName := strings.ToLower(textproto.CanonicalMIMEHeaderKey(name))
		if previous, exists := seenNames[canonicalName]; exists {
			return nil, fmt.Errorf("%w: %q and %q", ErrAmbiguousHeaders, previous, name)
		}
		seenNames[canonicalName] = name
		if _, ok := natsAuthHeaders[canonicalName]; ok {
			continue
		}
		copiedValues := append([]string(nil), values...)
		entries = append(entries, entry{name: canonicalName, values: copiedValues})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	var out bytes.Buffer
	for _, item := range entries {
		writeNATSHeaderFrame(&out, []byte(item.name))
		var count [8]byte
		binary.BigEndian.PutUint64(count[:], uint64(len(item.values)))
		_, _ = out.Write(count[:])
		for _, value := range item.values {
			writeNATSHeaderFrame(&out, []byte(value))
		}
	}
	return out.Bytes(), nil
}

func writeNATSHeaderFrame(out *bytes.Buffer, value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = out.Write(length[:])
	_, _ = out.Write(value)
}
