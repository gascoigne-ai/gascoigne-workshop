//go:build release

package main

// welcomeEveryStart is false in the release build: the welcome dialog opens
// once per version.
const welcomeEveryStart = false
