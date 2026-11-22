package app

import (
	"context"
	"errors"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"strings"

	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// pickGRPCMethod asks the server for its methods in the background and
// opens a menu of them; the chosen one becomes the request's method.
func (a *App) pickGRPCMethod(tab *reqTab) {
	// Proto files are read without a server; reflection asks one.
	if tab.Req.Proto.Kind() == request.SourceReflection && strings.TrimSpace(tab.Req.URL) == "" {
		a.toast("Enter the gRPC server first, for example grpc://localhost:50051", severity.Error)
		return
	}
	req := tab.Req.Expanded(a.varsFor(tab.Req))
	x, y := a.ctx.In.X, a.ctx.In.Y
	a.status = "Listing methods of " + req.URL
	timeout := a.timeoutFor(req) // read here: the goroutine must not touch App fields
	a.bus.Run(context.Background(), func(ctx context.Context) bus.Msg {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		methods, err := a.listGRPCMethods(ctx, req)
		return func() { // runs on the UI goroutine
			if err != nil {
				a.toast("Listing methods failed: "+err.Error(), severity.Error)
				return
			}
			items := make([]MenuItem, 0, len(methods))
			for _, method := range methods {
				items = append(items, MenuItem{Label: method, Run: func() { editRequest(tab, editor.MethodChosen{Method: method}) }})
			}
			a.openMenu(x, y, items)
		}
	})
}

// grpcMethodLabel is what the method chip of a gRPC tab shows, and in
// which color: the method, or a muted prompt to pick one.
func grpcMethodLabel(tab *reqTab) (string, w.Token) {
	if tab.Req.Method == "" {
		return "Pick method...", w.Muted
	}
	return tab.Req.Method, w.Accent
}

// grpcMethodW is the width of the method chip: wide, as method names are
// long.
const grpcMethodW = 200

// toggleStream switches a gRPC tab between single calls and a stream shown
// in the Messages pane. An open stream is closed first, since it belongs to
// the mode that opened it.
func (a *App) toggleStream(tab *reqTab) {
	a.disconnect(tab)
	tab.grpcStream = !tab.grpcStream
}

func (a *App) listGRPCMethods(ctx context.Context, req request.Request) ([]string, error) {
	if a.grpcMethods == nil {
		return nil, errors.New("no gRPC client")
	}
	return a.grpcMethods(ctx, req)
}
