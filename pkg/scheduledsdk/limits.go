package scheduledsdk

import (
	"encoding/json"
	"fmt"
)

const (
	MaxExecutorPayloadBytes       = 256 << 10
	MaxExecutionMetadataJSONBytes = 64 << 10
)

// ValidateExecutionMetadata applies one serialized-size boundary at both the
// task write boundary and worker consumption. Encoded size matters because
// JSON escaping can expand control characters substantially.
func ValidateExecutionMetadata(metadata map[string]string) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("%w: encode metadata: %v", ErrInvalidRequest, err)
	}
	if len(data) > MaxExecutionMetadataJSONBytes {
		return fmt.Errorf("%w: metadata exceeds %d encoded bytes", ErrInvalidRequest, MaxExecutionMetadataJSONBytes)
	}
	return nil
}
