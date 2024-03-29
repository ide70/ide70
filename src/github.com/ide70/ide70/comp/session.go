package comp

import (
	"crypto/rand"
	"github.com/kjk/betterguid"
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
	passItems map[string]*PassItem
	accessPrefix string
}

type PassItem struct {
	Created      time.Time
	params       map[string]interface{}
	parentUnitId string
}

type PassContext struct {
	Params       map[string]interface{}
	ParentUnitId string
}

func NewSession(accessPrefix string) *Session {
	now := time.Now()
	return &Session{ID: genID(), IsNew: true, Created: now, accessed: now, Timeout: 30 * time.Minute, rwMutex: &sync.RWMutex{}, UnitCache: NewUnitCache(), attrs: map[string]interface{}{}, passItems: map[string]*PassItem{}, accessPrefix:accessPrefix}
}

// Valid characters (bytes) to be used in session IDs
// Its length must be a power of 2.
const idChars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-_"
const idLength = 22
const AUTH_USER = "_authUser"
const AUTH_ROLE = "_authRole"
const passParamsMaxLifetimeSec = 120

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

func (s *Session) Access() {
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

func (s *Session) SetPassParameters(params map[string]interface{}, unit *UnitRuntime) string {
	s.cleanupPassParameters()
	passItem := &PassItem{}
	passItem.Created = time.Now()
	passItem.params = params
	passItem.parentUnitId = unit.getID()
	id := betterguid.New()
	s.passItems[id] = passItem
	return id
}

func (s *Session) GetPassParameters(id string) *PassContext {
	if id == "" {
		return &PassContext{map[string]interface{}{}, ""}
	}
	passItem := s.passItems[id]
	if passItem == nil {
		return &PassContext{map[string]interface{}{}, ""}
	}
	delete(s.passItems, id)
	return &PassContext{passItem.params, passItem.parentUnitId}
}

func (s *Session) cleanupPassParameters() {
	nowS := time.Now().Unix()
	for k, v := range s.passItems {
		if nowS-v.Created.Unix() > passParamsMaxLifetimeSec {
			delete(s.passItems, k)
		}
	}
}

func (sess *Session) DeleteUnit(unit *UnitRuntime) {
	delete(sess.UnitCache.ActiveUnits, unit.getID())
	logger.Debug("Active unints in session:", len(sess.UnitCache.ActiveUnits))
}

func (sess *Session) Accessible(action string) bool {
	return sess.accessPrefix == action
}

func (sess *Session) GetAccessPrefix() string {
	return sess.accessPrefix
}
