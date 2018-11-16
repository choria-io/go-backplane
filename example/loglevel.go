package main

import (
	"log"

	"github.com/choria-io/go-backplane/backplane"
)

// SetLogLevel implements backplane.LogLevelSetable
func (a *App) SetLogLevel(level backplane.LogLevel) {
	switch level {
	case backplane.DebugLevel:
		log.Printf("Setting log level to debug")
		a.config.LogLevel = "debug"
	case backplane.InfoLevel:
		log.Printf("Setting log level to info")
		a.config.LogLevel = "info"
	case backplane.WarnLevel:
		log.Printf("Setting log level to warning")
		a.config.LogLevel = "warning"
	case backplane.CriticalLevel:
		log.Printf("Setting log level to critical")
		a.config.LogLevel = "critical"
	default:
		log.Printf("Unknown LogLevel received - setting log level to debug")
		a.config.LogLevel = "debug"
	}
}

// GetLogLevel implements backplane.LogLevelSetable
func (a *App) GetLogLevel() backplane.LogLevel {
	switch a.config.LogLevel {
	case "debug":
		return backplane.DebugLevel
	case "info":
		return backplane.InfoLevel
	case "warning":
		return backplane.WarnLevel
	case "critical":
		return backplane.CriticalLevel
	}

	return backplane.DebugLevel
}
