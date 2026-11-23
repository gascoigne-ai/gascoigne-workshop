package sidebar

import (
	"fmt"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// benchSidebar is a sidebar over one open collection of rows rows (the root
// and rows-1 requests), expanded.
func benchSidebar(b testing.TB, rows int) (*core.Ctx, *Component) {
	b.Helper()
	col := &request.Collection{Folder: request.Folder{Name: "Big"}}
	for i := 0; i < rows-1; i++ {
		col.Folder.Requests = append(col.Folder.Requests, &request.Request{Name: fmt.Sprintf("r%d", i), Protocol: "rest", Method: "GET"})
	}
	fake := &fake{set: collections.Update(collections.Set{}, collections.Open{Collection: col})}
	ctx, side := newSidebar(b, fake)
	side.Tree().ExpandAll(&col.Folder)
	return ctx, side
}

// BenchmarkSidebarRows costs the two halves of a drawn frame of the tree by
// row count: one build and, laid out twice per frame, the layout.
func BenchmarkSidebarRows(b *testing.B) {
	for _, rows := range []int{200, 5000} {
		b.Run(fmt.Sprintf("rows=%d/build", rows), func(b *testing.B) {
			ctx, side := benchSidebar(b, rows)
			b.ReportAllocs()
			for b.Loop() {
				side.View(ctx, 900)
			}
		})
		b.Run(fmt.Sprintf("rows=%d/layout", rows), func(b *testing.B) {
			ctx, side := benchSidebar(b, rows)
			root := side.View(ctx, 900)
			b.ReportAllocs()
			for b.Loop() {
				Layout(ctx, root, sidebarW, 900)
			}
		})
	}
}
