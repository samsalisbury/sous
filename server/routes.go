package server

var (
	// SousRouteMap is the configuration of route for the application.
	SousRouteMap = RouteMap{
		{"gdm", "/gdm", &GDMResource{}},
		{"defs", "/defs", &StateDefResource{}},
		{"manifest", "/manifest", &ManifestResource{}},
		{"artifact", "/artifact", &ArtifactResource{}},
		{"status", "/status", &StatusResource{}},
		{"servers", "/servers", &ServerListResource{}},
	}
)
