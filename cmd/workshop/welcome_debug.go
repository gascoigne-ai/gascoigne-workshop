//go:build !release

package main

// welcomeEveryStart is true in the default (dev) build, so a run from an
// IDE always shows the welcome dialog; the release build shows it once
// per version.
const welcomeEveryStart = true
