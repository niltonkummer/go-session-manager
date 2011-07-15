package session

import (
	"crypto/rand"
	"fmt"
	"http"
	"log"
	"syscall"
	"sync"
	"time"
)

type Session struct {
	Id      string
	Value   interface{}
	expire  int64
	manager *SessionManager
	res     http.ResponseWriter
}

type SessionManager struct {
	sessionMap map[string]*Session
	onStart    func(*Session)
	onEnd      func(*Session)
	timeout    uint
	mutex      sync.RWMutex
}

func (session *Session) Abandon() {
	_, found := (*session.manager).sessionMap[session.Id]
	if found {
		(*session.manager).sessionMap[session.Id] = nil, false
	}
	if session.res != nil {
		session.res.Header().Set("Set-Cookie", "SessionId=; path=/;")
	}
}

func (session *Session) Cookie() string {
	tm := time.SecondsToUTC(session.expire)
	return fmt.Sprintf("SessionId=%s; path=/; expires=%s;", session.Id, tm.Format("Fri, 02-Jan-2006 15:04:05 -0700"))
}

func NewSessionManager(logger *log.Logger) *SessionManager {
	manager := new(SessionManager)
	manager.sessionMap = make(map[string]*Session)
	manager.timeout = 300
	go func(manager *SessionManager) {
		for {
			l := time.LocalTime().Seconds()
			for id, v := range (*manager).sessionMap {
				if v.expire < l {
					// expire
					if logger != nil {
						logger.Printf("Expired session(id:%s)", id)
					}
					f := (*manager).onEnd
					if f != nil {
						f((*manager).sessionMap[id])
					}
					(*manager).sessionMap[id] = nil, false
				}
			}
			syscall.Sleep(1000000000)
		}
	}(manager)
	return manager
}

func (manager *SessionManager) OnStart(f func(*Session)) { manager.onStart = f }
func (manager *SessionManager) OnEnd(f func(*Session))   { manager.onEnd = f }
func (manager *SessionManager) SetTimeout(t uint)        { manager.timeout = t }
func (manager *SessionManager) GetTimeout() uint         { return manager.timeout }

func (manager *SessionManager) GetSessionById(id string) *Session {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	if id == "" || !manager.Has(id) {
		b := make([]byte, 16)
		_, err := rand.Read(b)
		if err != nil {
			return nil
		}
		id = fmt.Sprintf("%x", b)
	}
	tm := time.SecondsToUTC(time.LocalTime().Seconds() + int64(manager.timeout))
	session, found := (*manager).sessionMap[id]
	if !found {
		session = &Session{id, nil, tm.Seconds(), manager, nil}
		(*manager).sessionMap[id] = session
		f := (*manager).onStart
		if f != nil {
			f(session)
		}
	} else {
		session.expire = tm.Seconds()
	}
	return session
}

func (manager *SessionManager) GetSession(res http.ResponseWriter, req *http.Request) *Session {
	if c, _ := req.Cookie("SessionId"); c != nil {
		session := manager.GetSessionById(c.Value)
		if res != nil {
			session.res = res
			res.Header().Set("Set-Cookie",
				fmt.Sprintf("SessionId=%s; path=/; expires=%s;",
					session.Id,
					time.SecondsToUTC(session.expire).Format(
						"Fri, 02-Jan-2006 15:04:05 -0700")))
		}
		return session
	}
	return manager.GetSessionById("")
}

func (manager *SessionManager) Has(id string) bool {
	_, found := (*manager).sessionMap[id]
	return found
}
