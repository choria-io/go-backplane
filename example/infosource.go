package main

// FactData implements backplane.InfoSource
func (a *App) FactData() interface{} {
	return a.config
}

// Version implements backplane.InfoSource
func (a *App) Version() string {
	return "0.0.1"
}
