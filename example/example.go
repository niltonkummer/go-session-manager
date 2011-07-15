package main

import (
	"github.com/mattn/go-session-manager"
	"http"
	"log"
	"os"
	"strings"
	"template"
)

const page = `
<!doctype html>
<html>
<meta charset="utf-8"/>
<body>
{.section session}
{.section Value}
Hi {@}.
<form method="post" action="/logout">
<input type="submit" name="method" value="logout" />
</form>
You will logout after 10 seconds. Then try to reload.
{.or}
<form method="post" action="/login">
<label for="name">Name:</label>
<input type="text" id="name" name="name" value="" />
<input type="submit" name="method" value="login" />
</form>
{.end}
{.end}
</body>
</html>
`

var fmap = template.FormatterMap{"html": template.HTMLFormatter}
var tmpl = template.MustParse(page, fmap)

func main() {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	manager := session.NewSessionManager(logger)
	manager.OnStart(func(session *session.Session) {
		println("started new session")
	})
	manager.OnEnd(func(session *session.Session) {
		println("abandon")
	})
	manager.SetTimeout(10)

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		session := manager.GetSession(w, req)
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, map[string]interface{}{"session": session})
	}))
	http.Handle("/login", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		name := strings.Trim(req.FormValue("name"), " ")
		if name != "" {
			logger.Printf("User \"%s\" login", name)

			// XXX: set user own object.
			manager.GetSession(w, req).Value = name
		}
		http.Redirect(w, req, "/", http.StatusFound)
	}))
	http.Handle("/logout", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if manager.GetSession(w, req).Value != nil {
			// XXX: get user own object.
			name := manager.GetSession(w, req).Value.(string)
			logger.Printf("User \"%s\" logout", name)
			manager.GetSession(w, req).Abandon()
		}
		http.Redirect(w, req, "/", http.StatusFound)
	}))
	err := http.ListenAndServe(":6061", nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
