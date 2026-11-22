package exampleserver

import (
	"net/http"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto/ws"
)

// echoWebSocket sends every text and binary message back until the client
// closes.
func echoWebSocket(writer http.ResponseWriter, request *http.Request) {
	conn, err := ws.Upgrade(writer, request)
	if err != nil {
		return // Upgrade has already answered the client
	}
	defer conn.Close(ws.CloseNormal, "")
	for {
		message, err := conn.Read()
		if err != nil || message.Type == ws.Close {
			return
		}
		if message.Type != ws.Text && message.Type != ws.Binary {
			continue
		}
		if conn.Write(message.Type, message.Data) != nil {
			return
		}
	}
}
