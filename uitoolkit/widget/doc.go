// Package widget holds the ready-made widgets built from view's elements:
// buttons, rows, tabs, menus, dialogs and toasts. Each one is a plain
// composition of view primitives, so it needs nothing from the runtime
// beyond what view exports. Like view, most callers dot-import it.
// Runnable examples: package uitoolkit/examples.
//
// # Finding your way
//
//	Buttons    Button PrimaryButton LinkButton IconButton Choice
//	Rows       ListRow FormRow CheckRow Note Toolbar
//	Tabs       Tabs ScrollTabs TabBar ActiveTab   (state: TabItem)
//	Menus      Menu OpenMenu CloseMenu MenuOpen PopupMenu PopupMenuLayer PopupMenuKeys NewMenu NextSelectable MenuWidth TextMenu   (state: MenuItem MenuState)
//	Dialogs    Dialog ResizableDialog DialogTitle MessageDialog   (state: DialogState DialogButton)
//	Toasts     Toasts   (state: ToastStack Toast)
//	Pictures   Image
//	Tables     Table   (state: TableSpec TableRow TableEdit ColumnWidths)   MinColumnPx
//	Resizing   Resizable   (state: ResizeState)
//	Splitting  Split Vertical AsRatio MinFirst MinSecond   (state: SplitState)
//	Sliders    Slider OnSlide OnSlideEnd SliderMid   (state: SliderState)
//
// The popup menu lives in core.Ctx.Menu: any element opens one with
// OpenMenu, and the window mounts PopupMenuLayer once and calls
// PopupMenuKeys each frame.
package widget
