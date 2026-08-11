package server

import (
	"encoding/json"
	"fmt"
	"io"
)

const maxAuthorityResponseBytes = 1 << 20

func decodeAuthorityResponse(body io.Reader, out interface{}) error {
	data, err := io.ReadAll(io.LimitReader(body, maxAuthorityResponseBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxAuthorityResponseBytes {
		return fmt.Errorf("authority response exceeds %d MiB limit", maxAuthorityResponseBytes>>20)
	}
	return json.Unmarshal(data, out)
}
