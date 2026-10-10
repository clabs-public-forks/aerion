package sync

import gosync "sync"

// folderChangeSeq counts, per folder, the engine's writes that change which
// messages or bodies a folder stores. Flag-only updates don't count.
type folderChangeSeq struct {
	mu  gosync.Mutex
	seq map[string]uint64
}

func (c *folderChangeSeq) bump(folderID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.seq == nil {
		c.seq = map[string]uint64{}
	}
	c.seq[folderID]++
}

func (c *folderChangeSeq) get(folderID string) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.seq[folderID]
}

// FolderChangeSeq returns a counter that moves whenever the engine stores or
// removes a message or a message body in folderID, so a caller can compare
// readings taken before and after a sync to see whether it changed anything.
func (e *Engine) FolderChangeSeq(folderID string) uint64 {
	return e.changeSeq.get(folderID)
}
