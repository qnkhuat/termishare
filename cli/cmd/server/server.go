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

type Room struct {
	ID    string
	conns []*websocket.Conn
	lock  sync.Mutex
}

func NewRoom(ID string) *Room {
	return &Room{
		ID: ID,
	}
}

func (ro *Room) String() string {
	return fmt.Sprintf("Room{%s, %d}", ro.ID, len(ro.conns))
}

func (ro *Room) removeConn(conn *websocket.Conn) {
	for i, c := range ro.conns {
		if c == conn {
			ro.lock.Lock()
			ro.conns = append(ro.conns[:i], ro.conns[i+1:]...)
			ro.lock.Unlock()
			log.Printf("%s - removed a connection", ro)
		}
	}
}

func (ro *Room) addConn(conn *websocket.Conn) {
	ro.lock.Lock()
	ro.conns = append(ro.conns, conn)
	ro.lock.Unlock()
	conn.SetCloseHandler(func(code int, text string) error {
		ro.removeConn(conn)
		return nil
	})

	log.Printf("%s - new connection", ro)

	// blocking
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			// TODO: need cleaner way to close it
			log.Printf("Failed to read message: %s. Closing", err)
			conn.Close()
			return
		}

		// Broadcast the message to everyone
		ro.Broadcast(msgType, msg, conn, false)
	}
}

func (ro *Room) Broadcast(msgType int, msg []byte, sender *websocket.Conn, self bool) {
	for _, c := range ro.conns {
		if self && c != sender {
			c.WriteMessage(msgType, msg)
		}
	}
}

type Server struct {
	addr   string
	server *http.Server
	rooms  map[string]*Room
	lock   sync.Mutex
}

func New(addr string) *Server {
	return &Server{
		addr:  addr,
		rooms: make(map[string]*Room),
	}
}

// Get room if existed, create then returns if not
func (sv *Server) getRoom(ID string) (*Room, error) {
	if room, ok := sv.rooms[ID]; ok {
		return room, fmt.Errorf("Room existed: %s", ID)
	} else {
		room := NewRoom(ID)
		log.Printf("Created a new room with ID: %s", ID)
		sv.lock.Lock()
		sv.rooms[ID] = room
		sv.lock.Unlock()
		return room, nil
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
	vars := mux.Vars(r)
	roomID := vars["roomID"]
	room, err := sv.getRoom(roomID)
	room.addConn(conn)
}

func (sv *Server) Start() {
	log.Printf("Serving at %s", sv.addr)
	fmt.Printf("Serving at %s\n", sv.addr)

	router := mux.NewRouter()
	router.Use(CORS)

	router.HandleFunc("/", handleHealth)
	router.HandleFunc("/ws/{roomID}", sv.WShandler)

	sv.server = &http.Server{Addr: sv.addr, Handler: router}
	err := sv.server.ListenAndServe()

	if err != nil {
		log.Printf("Failed to start server: %s", err)
	}
	log.Printf("Stop!")
}

func main() {
	var host = flag.String("host", "localhost:3000", "Host address to serve server")

	flag.Parse()

	logging.Config("/tmp/termishare.log", "SERVER: ")
	s := New(*host)
	s.Start()
}
