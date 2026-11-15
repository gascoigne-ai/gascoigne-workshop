// Package userdir names the program's directory inside the user config
// directory, where settings, state, history, themes and saved collections
// live. The name is per build (decision 20), so it is the one place every
// adapter that writes there takes it from.
package userdir

// releaseName is the directory of the release build; the default build
// appends a suffix to it, so the two never share data.
const releaseName = "gascoigne-workshop"
