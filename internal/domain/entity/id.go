package entity

import (
	"fmt"
	"sync/atomic"
	"time"
)

var idCounter int64

// generateID 生成唯一ID
func generateID() string {
	id := atomic.AddInt64(&idCounter, 1)
	return fmt.Sprintf("id_%d_%d", time.Now().UnixNano(), id)
}
