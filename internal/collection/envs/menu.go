package envs

import "github.com/gascoigne-ai/gascoigne-workshop/internal/request"

// Action is what a row of the chip's menu does.
type Action int

// The menu's actions. Select, Rename and Delete act on Row.Env.
const (
	Select Action = iota // make Row.Env the active environment (-1 for none)
	New                  // ask for a name and add an environment
	Rename               // ask for a new name for the active environment
	Delete               // ask, then delete the active environment
	Separator
)

// Row is one row of the chip's menu.
type Row struct {
	Label    string
	Action   Action
	Env      int
	Disabled bool
	Checked  bool // the active environment; the view marks it
}

// Menu is the rows of the chip's menu for the collection: the choices
// (the selected one checked), then New, Rename and Delete, which need an
// environment to be selected.
func Menu(sel Selection, col *request.Collection) []Row {
	rows := []Row{{Label: "No environment", Checked: sel.Index < 0, Action: Select, Env: -1}}
	for i, env := range col.Envs {
		rows = append(rows, Row{Label: env.Name, Checked: i == sel.Index, Action: Select, Env: i})
	}
	return append(rows,
		Row{Action: Separator},
		Row{Label: "New environment...", Action: New},
		Row{Label: "Rename...", Action: Rename, Env: sel.Index, Disabled: sel.Index < 0},
		Row{Label: "Delete", Action: Delete, Env: sel.Index, Disabled: sel.Index < 0},
	)
}
