package proto

import "testing"

// Whatever bytes a server sends, deliver returns: it logs the message, never
// panics and never waits for anything.
func FuzzMCPDeliver(fuzz *testing.F) {
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":1,"result":` + toolsResult + `}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":2,"result":{"resources":[{"uri":"file:///a","name":"a"}]}}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":3,"result":{"prompts":[{"name":"p","arguments":[{"name":"x","required":true}]}]}}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":4,"method":"sampling/createMessage","params":{}}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":5,"error":{"code":-32601,"message":"method not found"}}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"t","inputSchema":true}]}}`))
	fuzz.Add([]byte(`not json`))
	fuzz.Fuzz(func(t *testing.T, message []byte) {
		session := &mcpSession{eventLog: newTestLog(64), tr: closedTransport{}, waiters: map[int64]chan rpcAnswer{}}
		session.deliver(message)
	})
}
