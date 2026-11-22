package request

// OpenCollection is one collection the window had open: the folder it is
// saved in and the name of its active environment ("" for none). An entry
// without a folder stands for the collection the program starts with when
// none is saved: the built-in sample or the client config.
//
// It lives in this base package because the settings schema persists it and
// the collection set builds it; neither may import the other, so a shared
// base is the fix. Its JSON keys are part of the settings file (see
// persistence.md), so they do not change here.
type OpenCollection struct {
	Folder      string `json:"folder,omitempty"`
	Environment string `json:"environment,omitempty"`
}
