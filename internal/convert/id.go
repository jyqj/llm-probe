package convert

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

var counter uint64

// GenerateMessageID creates a unique message ID in the style of "msg_01XFDUDYJgAACzvnptvVoYEL".
func GenerateMessageID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp + counter
		n := atomic.AddUint64(&counter, 1)
		return fmt.Sprintf("msg_gw_%d_%d", time.Now().UnixNano(), n)
	}
	return "msg_gw_" + hex.EncodeToString(b)
}
