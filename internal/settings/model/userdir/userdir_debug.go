//go:build !release

package userdir

// Name is the default (dev) build's directory: a run from an IDE behaves
// like an install, remembering its own collections, without reading or
// overwriting the release build's.
const Name = releaseName + "-dev"
