// Package userdir names the program's directory inside the user config
// directory, where settings, state, history, themes and saved collections
// live. The name is per build, so it is the one place every adapter that
// writes there takes it from.
//
// The default (dev) build has a directory of its own, gascoigne-workshop-dev,
// so it reopens what it remembered, like an install, and never touches the
// release build's data (the user's choice). A directory without the suffix
// left by an earlier dev build is left alone: no migration, no warning.
package userdir

// releaseName is the directory of the release build; the default build
// appends a suffix to it, so the two never share data.
const releaseName = "gascoigne-workshop"
