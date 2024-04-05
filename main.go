package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type Message struct {
	from    string
	payload []byte
}

type Server struct {
	listenAddr string
	ln         net.Listener
	quitch     chan struct{}
	msgch      chan Message
	logFile    *os.File
}

func NewServer(listenAddr string) *Server {

	logFile, err := os.OpenFile("data.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	return &Server{
		listenAddr: listenAddr,
		quitch:     make(chan struct{}),
		msgch:      make(chan Message, 10),
		logFile:    logFile,
	}
}

func (s *Server) logMessage(message Message) {
	timestamp := time.Now().Format(time.RFC3339)
	logEntry := fmt.Sprintf("Your message is: %s. Received time: %s\n", string(message.payload), timestamp)

	if _, err := s.logFile.WriteString(logEntry); err != nil {
		log.Printf("Error writing to log file: %v", err)
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()
	s.ln = ln

	go s.acceptLoop()

	<-s.quitch
	close(s.msgch)
	s.logFile.Close()

	return nil
}

func (s *Server) handleCommand(conn net.Conn, cmd string) {
	cmd = strings.TrimSpace(cmd)

	if cmd == "/join" {
		fmt.Printf("User %s joined the chat.\n", conn.RemoteAddr())
	}
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			continue
		}

		fmt.Println("new connection to the server:", conn.RemoteAddr())

		go s.readLoop(conn)
	}
}

func (s *Server) readLoop(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)

	for {
		msg, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("connection closed by client")
			} else {
				fmt.Println("read error:", err)
			}
			return
		}

		if strings.HasPrefix(msg, "/") {
			s.handleCommand(conn, msg)
			continue
		}

		s.msgch <- Message{
			from:    conn.RemoteAddr().String(),
			payload: []byte(msg),
		}
	}
}

func main() {
	server := NewServer(":3000")

	go func() {
		for msg := range server.msgch {
			fmt.Println("received message from connection:", msg.from, string(msg.payload))
			server.logMessage(msg)
		}
	}()

	log.Fatal(server.Start())
}
