package envs

import "github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"

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
}

// Menu is the rows of the chip's menu for the collection: the choices
// (the selected one ticked), then New, Rename and Delete, which need an
// environment to be selected.
func Menu(sel Selection, col *model.Collection) []Row {
	mark := func(on bool, label string) string {
		if on {
			return "✓ " + label
		}
		return "   " + label
	}
	rows := []Row{{Label: mark(sel.Index < 0, "No environment"), Action: Select, Env: -1}}
	for i, env := range col.Envs {
		rows = append(rows, Row{Label: mark(i == sel.Index, env.Name), Action: Select, Env: i})
	}
	return append(rows,
		Row{Action: Separator},
		Row{Label: "New environment...", Action: New},
		Row{Label: "Rename...", Action: Rename, Env: sel.Index, Disabled: sel.Index < 0},
		Row{Label: "Delete", Action: Delete, Env: sel.Index, Disabled: sel.Index < 0},
	)
}
