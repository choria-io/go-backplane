package main

import (
	"os"
)

// Shutdown implements bacplane.Stopable
func (a *App) Shutdown() {
	os.Exit(0)
}
