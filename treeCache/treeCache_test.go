package treeCache_test

import (
	"aiosystem-backend/pkg/ws/treeCache"
	"log"
	"testing"
)

func TestDestroy(t *testing.T) {
	c := treeCache.NewSyncCache()
	c.SubCache("room1").SubCache("conn1").Set("alive", 1)
	m := c.SubCache("room1").SubCache("conn1").GetAll()
	for key, value := range m {
		log.Println(key, value)
	}
	c.Destroy()
	m = c.GetAll()
	for key, value := range m {
		log.Println(key, value)
		t.Error("should not have any thing", key, value)
	}

}
