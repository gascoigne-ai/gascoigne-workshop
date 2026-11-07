// Command hello is the smallest program built on the toolkit: a window
// with a text field, a button and a list, written once in Go and once as
// a template. Run it with: go run ./uitoolkit/examples/hello
package main

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/component"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// hello is the state of the program; the view is a function of it.
type hello struct {
	ctx     *core.Ctx
	sampler core.Sampler
	name    TextField
	greeted []string
	w, h    int
}

// view is written with the Go DSL: options first, then children.
func (g *hello) view() Node {
	return Col(Grow(1), Pad(core.Gutter), Gap(8), Bg(core.Bg),
		Text("Hello, toolkit", Bold()),
		Row(H(core.RowH), Gap(6),
			Field(&g.name, Grow(1), Placeholder("Your name"), OnEnter(g.greet)),
			PrimaryButton("Greet", W(80), OnClick(g.greet)),
		),
		g.list(),
	)
}

// list is the same kind of view written as a template; the data it reads
// is the scope.
func (g *hello) list() Node {
	return component.View(`
		<col gap="4">
		  <caption>Greeted</caption>
		  <row for="who in greeted" h="22" gap="6">
		    <icon name="plus" color="muted"/>
		    <text>{{who}}</text>
		  </row>
		  <text small muted if="!any">Nobody yet</text>
		</col>`,
		component.Scope{"greeted": g.names(), "any": len(g.greeted) > 0})
}

func (g *hello) names() []string { return g.greeted }

func (g *hello) greet() {
	if g.name.Text != "" {
		g.greeted = append(g.greeted, fmt.Sprintf("Hello, %s!", g.name.Text))
		g.name = NewTextField("")
		g.name.Focused = true
	}
}

func (g *hello) Layout(w, h int) (int, int) { g.w, g.h = w, h; return w, h }

func (g *hello) Update() error {
	g.ctx.In = g.sampler.Read()
	root := g.view()
	Layout(g.ctx, root, g.w, g.h)
	Update(g.ctx, root)
	return nil
}

func (g *hello) Draw(screen *ebiten.Image) {
	root := g.view()
	Layout(g.ctx, root, g.w, g.h)
	Draw(g.ctx, root, screen)
}

func main() {
	ctx, err := core.NewCtx(theming.Dark())
	if err != nil {
		log.Fatal(err)
	}
	g := &hello{ctx: ctx, name: NewTextField("")}
	ebiten.SetWindowSize(480, 320)
	ebiten.SetWindowTitle("Hello, toolkit")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
