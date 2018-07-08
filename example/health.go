package main

type health struct {
	Configured bool
}

// HealthCheck implements backplane.HealthCheckable
func (a *App) HealthCheck() (result interface{}, ok bool) {
	r := &health{
		Configured: a.configured,
	}

	return r, a.configured
}
