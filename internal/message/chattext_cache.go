package message

import (
	"hash/maphash"
	"sync"
)

// chatTextCacheSize bounds the cache; a long thread reload stays well inside it.
const chatTextCacheSize = 1024

type chatTextEntry struct {
	sum  uint64
	chat ChatText
}

// ChatTextCache memoizes ExtractChatText per message, so conversation reloads
// skip re-parsing bodies that have not changed. It is safe for concurrent use.
type ChatTextCache struct {
	mu      sync.Mutex
	seed    maphash.Seed
	entries map[string]chatTextEntry
}

// NewChatTextCache returns an empty cache.
func NewChatTextCache() *ChatTextCache {
	return &ChatTextCache{seed: maphash.MakeSeed(), entries: make(map[string]chatTextEntry)}
}

// Extract returns ExtractChatText(bodyText, bodyHTML), reusing the cached
// result for messageID when the body is unchanged.
func (c *ChatTextCache) Extract(messageID, bodyText, bodyHTML string) ChatText {
	var h maphash.Hash
	h.SetSeed(c.seed)
	h.WriteString(bodyText)
	h.WriteByte(0)
	h.WriteString(bodyHTML)
	sum := h.Sum64()

	c.mu.Lock()
	e, ok := c.entries[messageID]
	c.mu.Unlock()
	if ok && e.sum == sum {
		return e.chat
	}

	chat := ExtractChatText(bodyText, bodyHTML)

	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.entries[messageID]; !exists && len(c.entries) >= chatTextCacheSize {
		// Evict an arbitrary entry; recency tracking is not worth its cost here.
		for id := range c.entries {
			delete(c.entries, id)
			break
		}
	}
	c.entries[messageID] = chatTextEntry{sum: sum, chat: chat}
	return chat
}
