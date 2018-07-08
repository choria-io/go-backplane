package main

// Pause implements backplane.Pausable
func (a *App) Pause() {
	a.paused = true
}

// Resume implements backplane.Pausable
func (a *App) Resume() {
	a.paused = false
}

// Flip implements backplane.Pausable
func (a *App) Flip() {
	a.paused = !a.paused
}

// Paused implements backplane.Pausable
func (a *App) Paused() bool {
	return a.paused
}
