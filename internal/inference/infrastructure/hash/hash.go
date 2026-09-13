package hash

import (
	"bytes"
	"hash/fnv"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

type Hasher struct{}

func NewHasher() Hasher {
	return Hasher{}
}

func (h Hasher) HashRequest(key string) uint64 {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		if buf.Cap() <= 2048 {
			bufferPool.Put(buf)
		}
	}()

	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(key))

	return hasher.Sum64()
}
