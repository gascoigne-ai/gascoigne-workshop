package component_test

import (
	"fmt"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/component"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// A template written inline in the method that feeds it.
func ExampleView() {
	status := "Ready"
	node := component.View(`
		<row h="28" pad="12 0" bg="frame">
		  <text small muted>{{status}}</text>
		  <text small if="saved">saved</text>
		</row>`,
		component.Scope{"status": status, "saved": true})
	fmt.Print(view.Dump(node))
	// Output:
	// flow h=28 padL=12 padR=12 bg=frame
	//   textB text="Ready" fg=muted small=true
	//   textB text="saved" small=true
}

// A repeated element reads each item of a list.
func ExampleParse() {
	t, err := component.Parse("tabs.vml", []byte(`
		<row gap="4">
		  <button for="tab in tabs" on-click="pick">{{tab.name}}</button>
		</row>`))
	if err != nil {
		panic(err)
	}
	node, err := t.Build(component.Scope{
		"tabs": []map[string]any{{"name": "Params"}, {"name": "Body"}},
		"pick": func() {},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(strings.Count(view.Dump(node), "text="), "labels")
	// Output: 2 labels
}

// Mistakes name the file and line, suggest the nearest name and list
// the valid ones.
func ExampleParse_errors() {
	_, err := component.Parse("bar.vml", []byte("<row>\n  <txt/>\n</row>"))
	fmt.Println(strings.SplitN(err.Error(), ";", 2)[0])
	// Output: bar.vml:2: unknown element <txt> (did you mean "text"?)
}
