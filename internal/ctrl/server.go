package ctrl

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
)

var ErrInvalidCommand = errors.New("invalid command")
var ErrInvalidArguments = errors.New("invalid arguments")
var MsgCommandExecutedSuccessfully = []byte("command executed successfully")

type CommandHandler func(args []string) error

type Server struct {
	Network  string
	Address  string
	commands map[string]CommandHandler
}

func NewServer(network, bindAddr string) *Server {
	return &Server{Network: network, Address: bindAddr, commands: map[string]CommandHandler{}}
}

func (server *Server) AddCommand(label string, handler CommandHandler) {
	server.commands[label] = handler
}

func (server *Server) Listen() error {
	if server.Network == "unix" && strings.HasSuffix(server.Address, ".sock") {
		os.Remove(server.Address)
	}

	listener, err := net.Listen(server.Network, server.Address)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		slog.Debug("Client connected to control server")

		go server.HandleConnection(conn)
	}
}

func (server *Server) HandleConnection(conn net.Conn) {
	defer slog.Debug("Client disconnected from control server")

	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			continue
		}

		message = message[:len(message)-1]
		if len(message) == 0 {
			continue
		}

		args := strings.Split(message, " ")

		slog.Info(
			"Executing command",
			"command", message,
		)

		if handler, ok := server.commands[args[0]]; ok {
			if err = handler(args[1:]); err == nil {
				conn.Write(MsgCommandExecutedSuccessfully)
				continue
			}
		} else {
			err = ErrInvalidCommand
		}

		slog.Warn(
			"Error occured while executing command",
			"command", message,
			"error", err,
		)

		conn.Write([]byte(err.Error()))
	}
}
