package ws

import (
	"fmt"
	"hash/fnv"
	"unsafe"
)

func hashModPtr(ptr unsafe.Pointer, mod int) uint32 {
	hasher := fnv.New32a()
	hasher.Write([]byte(fmt.Sprintf("%p", ptr)))
	hashValue := hasher.Sum32()
	return hashValue % uint32(mod)
}
