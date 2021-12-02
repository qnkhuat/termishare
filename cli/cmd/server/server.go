package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/qnkhuat/termishare/internal/cfg"
	"github.com/qnkhuat/termishare/internal/util"
	"github.com/qnkhuat/termishare/pkg/message"
	"log"
	"net/http"
	"sync"
	"time"
)

// upgrade an http request to websocket
var httpUpgrader = websocket.Upgrader{
	ReadBufferSize:  cfg.SERVER_READ_BUFFER_SIZE,
	WriteBufferSize: cfg.SERVER_WRITE_BBUFFER_SIZE,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Headers:", "*")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type Server struct {
	addr   string
	server *http.Server
	conns  []*websocket.Conn
	lock   sync.Mutex
}

func New(addr string) *Server {

	return &Server{
		addr: addr,
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	log.Printf("health check")
	fmt.Fprintf(w, "I'm fine: %s\n", time.Now().String())
}

func (sv *Server) WShandler(w http.ResponseWriter, r *http.Request) {
	conn, err := httpUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection : %s", err)
		return
	}
	sv.AddConn(conn)

	log.Printf("New connection")

	conn.WriteJSON(message.Wrapper{Type: "ngoc"})

	for {
		msg := message.Wrapper{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Failed to read message: %s", err)
			continue
		}
		log.Printf("Got a message: %v", msg)

		//// Broadcast the message to everyone
		go func() {
			for _, c := range sv.conns {
				if c != conn {
					c.WriteJSON(msg)
				}
			}
		}()
	}
}

func (sv *Server) AddConn(conn *websocket.Conn) {
	sv.lock.Lock()
	sv.conns = append(sv.conns, conn)
	sv.lock.Unlock()
}

func (sv *Server) Start() {
	log.Printf("Serving at %s", sv.addr)
	fmt.Printf("Serving at %s\n", sv.addr)

	router := mux.NewRouter()
	router.Use(CORS)

	router.HandleFunc("/", handleHealth)
	router.HandleFunc("/ws", sv.WShandler)

	sv.server = &http.Server{Addr: sv.addr, Handler: router}
	err := sv.server.ListenAndServe()
	util.Must(err, "Failed to start server")
}

func main() {
	s := New("localhost:3000")
	s.Start()
}
