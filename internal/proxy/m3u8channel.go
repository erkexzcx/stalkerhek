package proxy

import (
	"sync"
	"time"

	"github.com/erkexzcx/stalkerhek/pkg/stalker"
)

type m3u8Channel struct {
	Stalker *stalker.Channel // Reference to stalker's channel

	linkOneAtOnceMux sync.Mutex // To ensure that only one routine at once is accessing/updating link

	link    string
	linkMux sync.RWMutex

	linkCache    []byte
	linkCacheMux sync.RWMutex

	linkCacheCreated    time.Time
	linkCacheCreatedMux sync.RWMutex

	linkRoot    string
	linkRootMux sync.RWMutex

	sessionUpdated    time.Time
	sessionUpdatedMux sync.RWMutex
}

func (c *m3u8Channel) Link() string {
	c.linkMux.RLock()
	defer c.linkMux.RUnlock()
	return c.link
}

func (c *m3u8Channel) SetLink(s string) {
	c.linkMux.Lock()
	defer c.linkMux.Unlock()
	c.link = s
}

func (c *m3u8Channel) LinkCache() []byte {
	c.linkCacheMux.RLock()
	defer c.linkCacheMux.RUnlock()
	return c.linkCache
}

func (c *m3u8Channel) SetLinkCache(b []byte) {
	c.linkCacheMux.Lock()
	defer c.linkCacheMux.Unlock()
	c.linkCache = b
}

func (c *m3u8Channel) LinkCacheCreated() time.Time {
	c.linkCacheCreatedMux.RLock()
	defer c.linkCacheCreatedMux.RUnlock()
	return c.linkCacheCreated
}

func (c *m3u8Channel) SetLinkCacheCreatedNow() {
	c.linkCacheCreatedMux.Lock()
	defer c.linkCacheCreatedMux.Unlock()
	c.linkCacheCreated = time.Now()
}

func (c *m3u8Channel) LinkRoot() string {
	c.linkRootMux.RLock()
	defer c.linkRootMux.RUnlock()
	return c.linkRoot
}

func (c *m3u8Channel) SetLinkRoot(s string) {
	c.linkRootMux.Lock()
	defer c.linkRootMux.Unlock()
	c.linkRoot = s
}

func (c *m3u8Channel) SessionUpdated() time.Time {
	c.sessionUpdatedMux.RLock()
	defer c.sessionUpdatedMux.RUnlock()
	return c.sessionUpdated
}

func (c *m3u8Channel) SetSessionUpdatedNow() {
	c.sessionUpdatedMux.Lock()
	defer c.sessionUpdatedMux.Unlock()
	c.sessionUpdated = time.Now()
}

// ----------

func (c *m3u8Channel) SessionValid() bool {
	s := c.SessionUpdated()
	if time.Since(s).Seconds() > 30 || s.IsZero() {
		return false
	}
	return true
}

func (c *m3u8Channel) LinkCacheValid() bool {
	s := c.LinkCacheCreated()
	if time.Since(s).Seconds() > 2 || s.IsZero() {
		return false
	}
	return true
}

func (c *m3u8Channel) UpdateLink() error {
	newLink, err := c.Stalker.NewLink()
	if err != nil {
		return err
	}
	c.SetLink(newLink)
	c.SetLinkRoot(deleteAfterLastSlash(newLink))
	c.SetSessionUpdatedNow()
	return nil
}

func getChannel(title string) (*m3u8Channel, bool) {
	if c, ok := channels[title]; ok {
		return c, true
	}
	if c, ok := stalkerChannels[title]; ok {
		channels[title] = &m3u8Channel{Stalker: c}
		return channels[title], true
	}
	return nil, false
}

func (c *m3u8Channel) newRedirectedLink(s string) {
	c.SetLink(s)
	c.SetLinkRoot(deleteAfterLastSlash(s))
}
