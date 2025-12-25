package router

import "github.com/relaypoint/relaypoint/internal/config"

type Route struct {
	Name       string
	Host       string
	Path       string
	Pattern    string
	Methods    map[string]bool
	Upstream   string
	StripPath  bool
	Headers    map[string]string
	RateLimit  *config.RouteRateLimit
	PathParams map[string]string
}

type Router struct {
	routes       []*routeEntry
	defaultRoute *routeEntry
}

type routeEntry struct {
	route      *Route
	segments   []segment
	isWildcard bool
	priority   int
}

type segment struct {
	value   string
	isParam bool
	isWild  bool
}
