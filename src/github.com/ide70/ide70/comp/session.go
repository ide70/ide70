package comp

import (
	"crypto/rand"
	"sync"
	"time"
)

type Session struct {
	ID        string                 // ID of the session
	IsNew     bool                   // Tells if the session is new
	Created   time.Time              // Creation time
	accessed  time.Time              // Last accessed time
	Timeout   time.Duration          // Session timeout
	rwMutex   *sync.RWMutex          // RW mutex to synchronize session (and related Window and component) access
	attrs     map[string]interface{} // Session attributes
	UnitCache *UnitCache
}

func NewSession() *Session {
	now := time.Now()
	return &Session{ID: genID(), IsNew: true, Created: now, accessed: now, Timeout: 30 * time.Minute, rwMutex: &sync.RWMutex{}, UnitCache: NewUnitCache(), attrs: map[string]interface{}{}}
}

// Valid characters (bytes) to be used in session IDs
// Its length must be a power of 2.
const idChars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-_"
const idLength = 22
const AUTH_USER = "_authUser"
const AUTH_ROLE = "_authRole"

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

func (s *Session) Attr(name string) interface{} {
	return s.attrs[name]
}

func (s *Session) AttrString(name string) string {
	val, has := s.attrs[name]
	if !has {
		return ""
	}
	return val.(string)
}

func (s *Session) HasAttr(name string) bool {
	_, has := s.attrs[name]
	return has
}

func (s *Session) SetAttr(name string, value interface{}) {
	if value == nil {
		delete(s.attrs, name)
	} else {
		s.attrs[name] = value
	}
}

func (s *Session) RwMutex() *sync.RWMutex {
	return s.rwMutex
}

func (s *Session) SetAuthUser(userName string) {
	s.SetAttr(AUTH_USER, userName)
}

func (s *Session) SetAuthRole(role string) {
	s.SetAttr(AUTH_ROLE, role)
}

func (s *Session) AuthUser() string {
	return s.AttrString(AUTH_USER)
}

func (s *Session) AuthRole() string {
	return s.AttrString(AUTH_ROLE)
}

func (s *Session) IsAuthenticated() bool {
	return s.HasAttr(AUTH_USER) 
}

func (s *Session) ClearAuthentication() {
	s.SetAttr(AUTH_USER, nil)
	s.SetAttr(AUTH_ROLE, nil)
}
