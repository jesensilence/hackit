package main

import (
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func showList(m *Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fixCSR(w)
		writeJSON(w, 200, m.list())
	}
}

type ReactRouter struct {
	fs    http.Handler
	other *mux.Router
}

func (rr ReactRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path

	var m mux.RouteMatch
	if rr.other.Match(r, &m) {
		rr.other.ServeHTTP(w, r)
		return
	}

	if _, err := os.Stat("./ui/build/" + p); err != nil {
		bs, _ := ioutil.ReadFile("./ui/build/index.html")
		w.WriteHeader(200)
		w.Write(bs)
		return
	} else {
		rr.fs.ServeHTTP(w, r)
	}
}

func makeId(v interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fixCSR(w)
		writeJSON(w, 200, v)
	}
}

func UIServer(sshAddr string, addr string, m *Manager) error {
	// See https://github.com/codegangsta/gin for get to known PORT environment.
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}

	r := mux.NewRouter()
	r.HandleFunc("/connectTTY/{uuid:[a-z0-9-]+}", m.HandleConnectTTY)
	r.HandleFunc("/connectChat/{uuid:[a-z0-9-]+}", m.HandleConnectChat)

	r.HandleFunc("/ssh_info", makeId(sshAddr))

	log.Printf("Listening http on %s\n", addr)
	return http.ListenAndServe(addr, ReactRouter{
		fs:    http.FileServer(http.Dir("./ui/build/")),
		other: r,
	})
}

func (m *Manager) HandleConnectTTY(w http.ResponseWriter, r *http.Request) {
	fixCSR(w)

	conn := m.findConnection(r)
	if conn == nil {
		writeJSON(w, 403, "Invalid magic key")
		return
	}

	ws, err := WSU().Upgrade(w, r, nil)
	if err != nil {
		writeJSON(w, 501, err.Error())
		return
	}

	done := make(chan struct{})
	go wsPing(ws, done)

	go func() {
		conn.Start(ws)
		done <- struct{}{}
		ws.Close()
		log.Println("Close ws...")
	}()

	log.Println("End of request ws")
}

func (m *Manager) findConnection(r *http.Request) *HackerConn {
	m.RLock()
	defer m.RUnlock()
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	return m.conns[uuid]
}

func (m *Manager) HandleConnectChat(w http.ResponseWriter, r *http.Request) {
	fixCSR(w)

	conn := m.findConnection(r)
	if conn == nil {
		writeJSON(w, 403, "Invalid magic key")
		return
	}

	// setup websocket
	ws, err := WSU().Upgrade(w, r, nil)
	if err != nil {
		writeJSON(w, 501, err)
		return
	}
	conn.SetupChat(ws)
	return
}
