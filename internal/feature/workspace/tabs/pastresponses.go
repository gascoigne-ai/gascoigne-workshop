package tabs

import (
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

// PastLimit is how many earlier responses a tab keeps for its History menu.
const PastLimit = 10

// PastResponse is an earlier response of a tab and when it was replaced.
type PastResponse struct {
	Resp *request.Response
	At   time.Time
}

// Remember puts the response a new one replaces into the history, newest
// first. Failed requests are not kept: there is nothing to look at.
func (tab *Tab) Remember(old *request.Response, at time.Time) {
	if old == nil || old.Err != "" {
		return
	}
	tab.Earlier = append([]PastResponse{{Resp: old, At: at}}, tab.Earlier...)
	if len(tab.Earlier) > PastLimit {
		tab.Earlier = tab.Earlier[:PastLimit]
	}
}

// Recall makes the response of history entry i the shown one; the shown one
// goes into the history in its place, so nothing is lost. It reports
// whether there was such an entry.
func (tab *Tab) Recall(index int, at time.Time) bool {
	if index < 0 || index >= len(tab.Earlier) {
		return false
	}
	picked := tab.Earlier[index].Resp
	if tab.Resp != nil && tab.Resp.Err == "" {
		tab.Earlier[index] = PastResponse{Resp: tab.Resp, At: at}
	} else {
		tab.Earlier = append(tab.Earlier[:index], tab.Earlier[index+1:]...)
	}
	tab.Resp = picked
	return true
}
