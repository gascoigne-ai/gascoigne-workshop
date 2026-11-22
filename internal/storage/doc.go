// Package storage holds the plumbing every adapter that persists a file
// shares: where the program's files live, and how a saved JSON file is
// versioned. It is one package because both are used together by the
// adapters of four features (collections, history, settings, workspace) and
// neither has a reason to change apart from the other.
//
// Location (userdir.go, files.go): the program's directory inside the user
// config directory, where settings, state, history, themes and saved
// collections live. The name is per build, so it is the one place every
// adapter that writes there takes it from. The default (dev) build has a
// directory of its own, gascoigne-workshop-dev, so it reopens what it
// remembered, like an install, and never touches the release build's data
// (the user's choice). A directory without the suffix left by an earlier dev
// build is left alone: no migration, no warning.
//
// Versioning (fileversion.go): reads the version of a saved JSON file and
// picks the reader of that version from a table. Every file the workshop
// writes carries "version" as its first key; a file without one is version
// 0, the layout written before versions existed. The version is per file,
// not per collection, and lives in each adapter's wire type, not the model;
// architecture/persistence.md tables every file and what a newer version
// does (the user's answers).
package storage
