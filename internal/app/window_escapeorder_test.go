package app

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/form"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/overlay"
	kvlogic "github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// pressEscapeFor presses Escape once and checks that layer was the top one
// and is closed now, with every layer under it still open.
func pressEscapeFor(t *testing.T, a *App, layer overlay.Layer) {
	t.Helper()
	before := a.openLayers()
	if top, _ := before.Top(); top != layer {
		t.Fatalf("the top layer is %d, want %d", top, layer)
	}
	a.Step(core.Input{X: 5000, Y: 5000, Escape: true})
	after := a.openLayers()
	if after[layer] {
		t.Fatalf("Escape did not close layer %d", layer)
	}
	for under := layer + 1; under < overlay.Request; under++ {
		if before[under] != after[under] {
			t.Fatalf("closing layer %d changed layer %d", layer, under)
		}
	}
}

// Escape closes one layer per press, the one drawn on top first, whatever
// the layers belong to: the overlays, the table, the find bar, the request.
func TestEscapeClosesOneLayerPerPressInOrder(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	a.files = fakeDialog{suggest: "zenity"} // no dialog program, so the in-app browser opens
	tab := a.cur()
	tab.Req.Params = []request.KV{{Name: "q", Value: "1", Enabled: true}}
	a.openMenu(10, 10, []widget.MenuItem{{Label: "One", Run: func() {}}})
	a.confirm("Sure?", "body", "OK", func() {})
	a.openTransferPicker(request.Of(a.cols.Set().Entries[0].Collection.Requests[0]), transfer.Move)
	a.pickFile("Pick", filedialog.OpenFile, "", func(string) {})
	a.over.Calls().Open()
	a.over.Settings.Show()
	a.over.Settings.Form.Families = testFamilies(t)
	a.over.Settings.Form.OpenFontList(a.ctx, a.over.Settings, form.FontUI)

	for _, layer := range []overlay.Layer{overlay.ContextMenu, overlay.Dialog, overlay.DestPicker, overlay.FileBrowser,
		overlay.CallLog, overlay.FontList, overlay.Settings} {
		pressEscapeFor(t, a, layer)
	}

	a.resp.OpenFind() // opening the find bar ends a cell edit, so the edit comes after
	a.kv.BeginEdit(kvlogic.Table{Tab: tab.Tab, Kind: kvlogic.Params}, 0, 1)
	pressEscapeFor(t, a, overlay.CellEdit)
	pressEscapeFor(t, a, overlay.FindBar)
	if top, _ := a.openLayers().Top(); top != overlay.Request {
		t.Fatalf("the request is the last resort and always open, top is %d", top)
	}
}
