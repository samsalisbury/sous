package server

var (
	// SousRouteMap is the configuration of route for the application
	SousRouteMap = RouteMap{
		{"gdm", "/gdm", &GDMResource{}},
		{"manifest", "/manifest", &ManifestResource{}},
	}
)
