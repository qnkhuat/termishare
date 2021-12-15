package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/qnkhuat/termishare/internal/cfg"
	"github.com/qnkhuat/termishare/pkg/logging"
	"github.com/qnkhuat/termishare/pkg/message"
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
	log.Printf("Health check")
	fmt.Fprintf(w, "I'm fine: %s\n", time.Now().String())
}

func (sv *Server) WShandler(w http.ResponseWriter, r *http.Request) {
	conn, err := httpUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection : %s", err)
		return
	}
	sv.AddConn(conn)

	conn.SetCloseHandler(func(code int, text string) error {
		for i, c := range sv.conns {
			if c == conn {
				log.Printf("Removing a connection due closed")
				sv.lock.Lock()
				sv.conns = append(sv.conns[:i], sv.conns[i+1:]...)
				sv.lock.Unlock()
			}
		}
		return nil
	})

	log.Printf("New connection - %d", len(sv.conns))

	for {
		msg := message.Wrapper{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			// TODO: need cleaner way to close it
			log.Printf("Failed to read message: %s. Closing", err)
			conn.Close()
			return
		}

		// Broadcast the message to everyone
		for _, c := range sv.conns {
			if c != conn {
				c.WriteJSON(msg)
			}
		}
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

	if err != nil {
		log.Printf("Failed to start server: %s", err)
	}
}

func main() {
	var host = flag.String("host", "localhost:3000", "Host address to serve server")

	flag.Parse()

	logging.Config("/tmp/termishare.log", "SERVER: ")
	s := New(*host)
	s.Start()
}
