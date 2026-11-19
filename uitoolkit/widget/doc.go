// Package widget holds the ready-made widgets built from view's elements:
// buttons, rows, tabs, menus, dialogs and toasts. Most are compositions of
// view primitives; those with their own drawing, input or layout (scrolling,
// splitting, text editing, sliders, tables) are elements built with
// view.Custom. Either way they need nothing from the runtime beyond what
// view exports. Like view, most callers dot-import it.
// Runnable examples: package uitoolkit/examples.
//
// # Finding your way
//
//	Buttons    Button PrimaryButton LinkButton IconButton Choice
//	Rows       ListRow FormRow CheckRow Note Toolbar
//	Tabs       Tabs ScrollTabs TabBar ActiveTab   (state: TabItem)
//	Menus      Menu OpenMenu CloseMenu MenuOpen PopupMenu PopupMenuLayer PopupMenuKeys NewMenu NextSelectable MenuWidth TextMenu   (state: MenuItem MenuState)
//	Dialogs    Dialog ResizableDialog DialogTitle DialogSubtitle DialogToolbar MessageDialog   (state: DialogState DialogButton)
//	Toasts     Toasts   (state: ToastStack Toast ToastKind)
//	Pictures   Image RasterizeSVG
//	Inputs     Field Area Code   (state: TextField TextArea CodeView Line Match Span; NewTextField NewTextArea)   Placeholder FieldPad Segments Wrap OnEnter OnTab   (OnChange is view's)
//	Tooltips   TooltipLayer   (state: TooltipState)   (the Tooltip option is view's)
//	Icons      Icon RegisterIcons IconNames HasIcon   CrossIcon TriangleIcon GripIcon DotsIcon GearIcon TrashIcon PlusIcon CopyIcon FolderIcon SearchIcon PromptIcon ChevronUpIcon ChevronDownIcon
//	Checkboxes Checkbox
//	Tables     Table   (state: TableSpec TableRow TableEdit ColumnWidths)   MinColumnPx
//	Resizing   Resizable   (state: ResizeState)
//	Scrolling  Scroll HScroll   (state: ScrollState)   (the KeepInView option is view's)
//	Splitting  Split Vertical AsRatio MinFirst MinSecond   (state: SplitState)
//	Sliders    Slider OnSlide OnSlideEnd SliderMid   (state: SliderState)
//
// The popup menu lives in core.Ctx.Menu: any element opens one with
// OpenMenu, and the window mounts PopupMenuLayer once and calls
// PopupMenuKeys each frame.
package widget
