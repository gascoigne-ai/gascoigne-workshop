// Package openset is the record of one collection the window had open. It is
// a kernel leaf because the settings schema persists it and the collection
// set builds it; neither may import the other. Its JSON keys are part of the
// settings file (see persistence.md), so they do not change here.
package openset

// OpenCollection is one collection the window had open: the folder it is
// saved in and the name of its active environment ("" for none). An entry
// without a folder stands for the collection the program starts with when
// none is saved: the built-in sample or the client config.
type OpenCollection struct {
	Folder      string `json:"folder,omitempty"`
	Environment string `json:"environment,omitempty"`
}
