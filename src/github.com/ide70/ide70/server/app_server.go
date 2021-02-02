package server

import (
	"net/http"
	"time"
	//	"net/url"
	"fmt"
	"github.com/ide70/ide70/comp"
	"github.com/ide70/ide70/util/log"
	"strconv"
	"strings"
	"sync"
)

// Internal path constants.
const (
	pathStatic     = "_static/"
	pathSessCheck  = "_sess_ch"
	pathUnitCreate = "uc" // path for unit create
	pathEvent      = "e"  // Window-relative path for sending events
	pathEventAsync = "ea" // Window-relative path for sending events
	pathRenderComp = "rc" // Window-relative path for rendering a component
)

// Parameters passed between the browser and the server.
const (
	paramEventType     = "et"   // Event type parameter name
	paramCompID        = "cid"  // Component id parameter name
	paramCompValue     = "cval" // Component value parameter name
	paramFocusedCompID = "fcid" // Focused component id parameter name
	paramMouseWX       = "mwx"  // Mouse x pixel coordinate (inside window)
	paramMouseWY       = "mwy"  // Mouse y pixel coordinate (inside window)
	paramMouseX        = "mx"   // Mouse x pixel coordinate (relative to source component)
	paramMouseY        = "my"   // Mouse y pixel coordinate (relative to source component)
	paramMouseBtn      = "mb"   // Mouse button
	paramModKeys       = "mk"   // Modifier key states
	paramKeyCode       = "kc"   // Key code
	paramScrollTop     = "sctp" // Scroll top
)

const sessidCookieName = "ide70-sessid"

type AppServer struct {
	App               *Application
	AppParams         *comp.AppParams
	Addr              string              // Server address
	Secure            bool                // Tells if the server is configured to run in secure (HTTPS) mode
	sessions          map[string]*Session // Sessions
	certFile, keyFile string              // Certificate and key files for secure (HTTPS) mode
	sessMux           sync.RWMutex        // Mutex to protect state related to session handling
	sessStop          chan struct{}
	httpServer        *http.Server
}

var logger = log.Logger{"server"}

func NewAppServer(addr string, secure bool) *AppServer {
	appServer := &AppServer{}
	appServer.Addr = addr
	appServer.Secure = secure
	appServer.sessions = map[string]*Session{}
	return appServer
}

func (s *AppServer) SetApplication(app *Application) {
	app.registerServer(s)
	s.AppParams = &comp.AppParams{
		PathStatic: app.Path + pathStatic,
		Path:       app.Path,
		RuntimeID:  fmt.Sprintf("%d", time.Now().Unix()),
	}
	s.App = app
}

func (s *AppServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc(s.App.Path, func(w http.ResponseWriter, r *http.Request) {
		s.serveHTTP(w, r)
	})

	mux.HandleFunc(s.App.Path+pathStatic, func(w http.ResponseWriter, r *http.Request) {
		s.serveStatic(w, r)
	})

	logger.Info("Starting GUI server on:", s.App.URLString)

	s.sessStop = make(chan struct{})
	go s.sessCleaner(s.sessStop)

	s.httpServer = &http.Server{Addr: s.Addr, Handler: mux}

	var err error
	if s.Secure {
		GenCerts()
		err = s.httpServer.ListenAndServeTLS(certFileName(), keyFileName())
	} else {
		err = s.httpServer.ListenAndServe()
	}

	return err
}

func (s *AppServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	logger.Info("Incoming:", r.URL.Path)

	//s.addHeaders(w)

	// Check session
	var sess *Session
	c, err := r.Cookie(sessidCookieName)
	if err == nil {
		s.sessMux.RLock()
		sess = s.sessions[c.Value]
		s.sessMux.RUnlock()
	}
	/*if sess == nil {
		sess = &s.sessionImpl
	}*/

	// Parts example: "/appname/winname/e?et=0&cid=1" => {"", "appname", "winname", "e"}
	// Parts example: "/appname/action/unitName" => {"", "appname", "action", "unitName1", "unigNameN"}
	parts := strings.Split(r.URL.Path, "/")

	// We have app name
	if len(parts) < 2 {
		// Missing app name from path
		http.NotFound(w, r)
		return
	}
	// Omit the first empty string and the app name
	parts = parts[2:]

	if len(parts) < 1 || parts[0] == "" {
		// Missing action name
		http.NotFound(w, r)
		return
	}

	action := parts[0]
	logger.Info("action:", action)

	switch action {
	case pathEvent:
		unitId := parts[1]
		if sess == nil {
			logger.Error("no session for event")
			http.NotFound(w, r)
			return
		}
		unit := sess.UnitCache.ActiveUnits[unitId]
		if unit == nil {
			logger.Error("no unit found by id:", unitId)
			http.NotFound(w, r)
			return
		}
		rwMutex := sess.rwMutex
		rwMutex.Lock()
		defer rwMutex.Unlock()
		s.handleEvent(sess, unit, w, r)
	default:
		unitName := strings.Join(parts, "/")

		if sess == nil {
			if unitName == s.App.Access.LoginUnit {
				sess = s.newSession(nil)
				s.addSessCookie(sess, w)
				logger.Info("session created:", sess.ID)
			} else {
				http.Error(w, "no session", http.StatusUnauthorized)
			}
		}

		logger.Info("new unit runtime...")
		unit := comp.InstantiateUnit(unitName, s.AppParams)
		if unit == nil {
			logger.Error("no unit found by name:", unitName)
			http.NotFound(w, r)
			return
		}
		logger.Info("instantiation finished")
		if sess != nil {
			sess.UnitCache.addUnit(unit)
			logger.Info("unit runtime cached in session")
		}
		logger.Info("unit runtime render start..")
		unit.Render(w)
		logger.Info("unit runtime rendered")
	}

	/*
		win := sess.WinByName(winName)
		// If not found and we're on an authenticated session, try the public window list
		if win == nil && sess.Private() {
			win = s.WinByName(winName) // Server is a Session, the public session
			if win != nil {
				// We're serving a public window, switch to public session here entirely
				sess = &s.sessionImpl
			}
		}

		// If still not found and no private session, try the session creator names
		if win == nil && !sess.Private() {
			if _, found := s.sessCreatorNames[winName]; found {
				sess = s.newSession(nil)
				s.addSessCookie(sess, w)
				// Search again in the new session as SessionHandlers may have added windows.
				win = sess.WinByName(winName)
			}
		}

		if win == nil {
			// Invalid window name, render an error message with a link to the window list
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			NewWriter(w).Writess("<html><body>Window for name <b>'", winName, `'</b> not found.</body></html>`)
			return
		}

		sess.access()

		var path string
		if len(parts) >= 2 {
			path = parts[1]
		}

		rwMutex := sess.rwMutex()
		switch path {
		case pathEvent:
			rwMutex.Lock()
			defer rwMutex.Unlock()

			s.handleEvent(sess, win, w, r)
		case pathEventAsync:
			s.handleEvent(sess, win, w, r)
		case pathRenderComp:
			rwMutex.RLock()
			defer rwMutex.RUnlock()

			// Render just a component
			s.renderComp(win, w, r)
		default:
			rwMutex.RLock()
			defer rwMutex.RUnlock()

			// Render the whole window
			win.RenderWin(NewWriter(w), s)
		}*/
}

func (s *AppServer) handleEvent(sess *Session, unit *comp.UnitRuntime, wr http.ResponseWriter, r *http.Request) {
	/*focCompID, err := AtoID(r.FormValue(paramFocusedCompID))
	if err == nil {
		win.SetFocusedCompID(focCompID)
	}*/

	compId, err := strconv.ParseInt(r.FormValue(paramCompID), 10, 64)
	if err != nil {
		logger.Error("Invalid component id")
		http.Error(wr, "Invalid component id!", http.StatusBadRequest)
		return
	}

	c := unit.CompRegistry[compId]
	if c == nil {
		logger.Error("Component not found:", compId)
		http.Error(wr, fmt.Sprint("Component not found: ", compId), http.StatusBadRequest)
		return
	}
	
	logger.Info("event, component found:", c)

	etype := r.FormValue(paramEventType)
	if etype == "" {
		http.Error(wr, "Invalid event type!", http.StatusBadRequest)
		return
	}
	logger.Info("Event from comp:", compId, " event:", etype)
	
	e := &comp.EventRuntime{TypeCode: etype}
	c.CompDef.EventsHandler.ProcessEvent(e)
	
	wr.Write([]byte(e.ResponseAction.Encode()))


	/*event := newEventImpl(EventType(etype), comp, s, sess, wr, r)
	shared := event.shared

	event.x = parseIntParam(r, paramMouseX)
	if event.x >= 0 {
		event.y = parseIntParam(r, paramMouseY)
		shared.wx = parseIntParam(r, paramMouseWX)
		shared.wy = parseIntParam(r, paramMouseWY)
		shared.mbtn = MouseBtn(parseIntParam(r, paramMouseBtn))
	} else {
		event.y, shared.wx, shared.wy, shared.mbtn = -1, -1, -1, -1
	}

	shared.modKeys = parseIntParam(r, paramModKeys)
	shared.keyCode = Key(parseIntParam(r, paramKeyCode))
	shared.scrollTop = parseIntParam(r, paramScrollTop)

	comp.preprocessEvent(event, r)

	// Dispatch event...
	comp.dispatchEvent(event)

	// Check if a new session was created during event dispatching
	if shared.session.New() {
		s.addSessCookie(shared.session, wr)
	}
	*/
	// ...and send back the result
	wr.Header().Set("Content-Type", "text/plain; charset=utf-8") // We send it as text
	//w := NewWriter(wr)
	/*hasAction := false

	if shared.applyToParent {
		w.Writevs(eraApplyToParent, strSemicol)
	}

	// If we reload, nothing else matters
	if shared.reload {
		hasAction = true
		w.Writevs(eraReloadWin, strComma, shared.reloadWin)
	} else {
		if len(shared.dirtyComps) > 0 {
			hasAction = true
			w.Writev(eraDirtyComps)
			for id := range shared.dirtyComps {
				w.Write(strComma)
				w.Writev(int(id))
			}
		}
		if shared.focusedComp != nil {
			if hasAction {
				w.Write(strSemicol)
			} else {
				hasAction = true
			}
			w.Writevs(eraFocusComp, strComma, int(shared.focusedComp.ID()))
			// Also register focusable comp at window
			win.SetFocusedCompID(shared.focusedComp.ID())
		}
		if shared.scrolledDownComp != nil {
			if hasAction {
				w.Write(strSemicol)
			} else {
				hasAction = true
			}
			w.Writevs(eraScrollDownComp, strComma, int(shared.scrolledDownComp.ID()), strComma, int(shared.targetScrollTop))
		}
		if len(shared.dirtyCompAttrs) > 0 {
			if hasAction {
				w.Write(strSemicol)
			} else {
				hasAction = true
			}
			w.Writev(eraDirtyAttrs)
			for id, compAttr := range shared.dirtyCompAttrs {
				w.Writevs(strComma, int(id), strComma, compAttr.attrName, strComma, compAttr.comp.CalculatedAttr(compAttr.attrName))
			}
		}
	}
	if !hasAction {
		w.Writev(eraNoAction)
	}*/
}

func (s *AppServer) serveStatic(w http.ResponseWriter, r *http.Request) {
	//s.addHeaders(w)

	// Parts example: "/appname/_static/gwu-0.8.0.js" => {"", "appname", "_gwu_static", "gwu-0.8.0.js"}
	parts := strings.Split(r.URL.Path, "/")

	if len(parts) < 3 {
		// Missing app name from path
		http.NotFound(w, r)
		return
	}
	// Omit the first empty string, app name and pathStatic
	parts = parts[3:]

	res := parts[0]
	if strings.HasSuffix(res, ".js") {
		w.Header().Set("Expires", time.Now().UTC().Add(72*time.Hour).Format(http.TimeFormat)) // Set 72 hours caching
		w.Header().Set("Content-Type", "application/x-javascript; charset=utf-8")
		http.ServeFile(w, r, "ide70/js/"+res)
		return
	}
	if strings.HasSuffix(res, ".css") {
		w.Header().Set("Expires", time.Now().UTC().Add(72*time.Hour).Format(http.TimeFormat)) // Set 72 hours caching
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		http.ServeFile(w, r, "ide70/css/"+res)
		return
	}

	http.NotFound(w, r)
}

func (s *AppServer) sessCleaner(stop chan struct{}) {
	sleep := 10 * time.Second
	for {
		select {
		case <-stop:
			logger.Info("stop all sessions")
			s.sessMux.Lock()
			for _, sess := range s.sessions {
				s.removeSession(sess)
			}
			//s.removeSHandlers()
			s.sessMux.Unlock()
			return
		case <-time.After(sleep):
			now := time.Now()

			s.sessMux.Lock()
			for _, sess := range s.sessions {
				if now.Sub(sess.Accessed()) > sess.Timeout {
					s.removeSession(sess)
				}
			}
			s.sessMux.Unlock()
		}
	}
}

func (s *AppServer) newSession(oldSess *Session) *Session {
	sess := newSession()

	// Store new session
	s.sessMux.Lock()
	s.sessions[sess.ID] = sess

	logger.Info("SESSION created:", sess.ID)

	// Notify session handlers
	//for _, handler := range s.sessionHandlers {
	//	handler.Created(sess)
	//}
	s.sessMux.Unlock()

	return sess
}

// removeSess2 removes (invalidates) the specified session.
// Only private sessions can be removed, calling this with the
// public session is a no-op.
// serverImpl.mux must be locked when this is called.
func (s *AppServer) removeSession(sess *Session) {
	logger.Info("SESSION removed:", sess.ID)

	// Notify session handlers
	//for _, handler := range s.sessionHandlers {
	//	handler.Removed(sess)
	//}
	delete(s.sessions, sess.ID)
}

func (s *AppServer) addSessCookie(sess *Session, w http.ResponseWriter) {
	// HttpOnly: do not allow non-HTTP access to it (like javascript) to prevent stealing it...
	// Secure: only send it over HTTPS
	// MaxAge: to specify the max age of the cookie in seconds, else it's a session cookie and gets deleted after the browser is closed.
	c := http.Cookie{
		Name: sessidCookieName, Value: sess.ID,
		Path:     s.App.URL.EscapedPath(),
		HttpOnly: true, Secure: s.Secure,
		MaxAge: 72 * 60 * 60, // 72 hours max age
	}
	http.SetCookie(w, &c)
	sess.IsNew = false
}
