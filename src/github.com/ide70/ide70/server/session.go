package server

import (
	"crypto/rand"
	"sync"
	"time"
)

type Session struct {
	ID        string        // ID of the session
	IsNew     bool          // Tells if the session is new
	Created   time.Time     // Creation time
	accessed  time.Time     // Last accessed time
	Timeout   time.Duration // Session timeout
	rwMutex   *sync.RWMutex // RW mutex to synchronize session (and related Window and component) access
	UnitCache *UnitCache
}

func newSession() *Session {
	now := time.Now()
	return &Session{ID: genID(), IsNew: true, Created: now, accessed: now, Timeout: 30 * time.Minute, rwMutex: &sync.RWMutex{}, UnitCache: newUnitCache()}
}

// Valid characters (bytes) to be used in session IDs
// Its length must be a power of 2.
const idChars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-_"
const idLength = 22

// genID generates a new session ID.
func genID() string {
	id := make([]byte, idLength)
	if _, err := rand.Read(id); err != nil {
		logger.Error("Failed to read from secure random: %v", err)
	}

	for i, v := range id {
		id[i] = idChars[v&byte(len(idChars)-1)]
	}
	return string(id)
}

func (s *Session) Accessed() time.Time {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()
	return s.accessed
}

func (s *Session) access() {
	s.rwMutex.Lock()
	s.accessed = time.Now()
	s.rwMutex.Unlock()
}
