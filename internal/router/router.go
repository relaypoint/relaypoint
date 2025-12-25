package router

import (
	"net/http"
	"sort"
	"strings"

	"github.com/relaypoint/relaypoint/internal/config"
)

// New creates a new router from configuration
func New(routes []config.Route) *Router {
	r := &Router{
		routes: make([]*routeEntry, 0, len(routes)),
	}

	for _, cfg := range routes {
		methods := make(map[string]bool)
		if len(cfg.Methods) == 0 {
			// Allow all methods by default
			methods["*"] = true
		} else {
			for _, m := range cfg.Methods {
				methods[strings.ToUpper(m)] = true
			}
		}

		route := &Route{
			Name:      cfg.Name,
			Host:      strings.ToLower(cfg.Host),
			Path:      cfg.Path,
			Pattern:   cfg.Path,
			Methods:   methods,
			Upstream:  cfg.Upstream,
			StripPath: cfg.StripPath,
			Headers:   cfg.Headers,
			RateLimit: cfg.RateLimit,
		}

		entry := &routeEntry{
			route:    route,
			segments: parseSegments(cfg.Path),
		}

		// Calculate priority (more specific = higher priority)
		entry.priority = calculatePriority(entry.segments)
		entry.isWildcard = hasWildcard(entry.segments)

		r.routes = append(r.routes, entry)
	}

	// Sort routes by priority (higher priority first)
	sort.Slice(r.routes, func(i, j int) bool {
		return r.routes[i].priority > r.routes[j].priority
	})

	return r
}

// parseSegments parses a path pattern into segments
func parseSegments(path string) []segment {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}

	parts := strings.Split(path, "/")
	segments := make([]segment, len(parts))

	for i, part := range parts {
		switch {
		case part == "*" || part == "**":
			segments[i] = segment{value: part, isWild: true}
		case strings.HasPrefix(part, ":"):
			segments[i] = segment{value: part[1:], isParam: true}
		case strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}"):
			segments[i] = segment{value: part[1 : len(part)-1], isParam: true}
		default:
			segments[i] = segment{value: strings.ToLower(part)}
		}
	}

	return segments
}

// calculatePriority calculates route priority
func calculatePriority(segments []segment) int {
	priority := len(segments) * 10

	for _, seg := range segments {
		if seg.isWild {
			priority -= 5
		} else if seg.isParam {
			priority -= 2
		} else {
			priority += 3
		}
	}

	return priority
}

func hasWildcard(segments []segment) bool {
	for _, seg := range segments {
		if seg.isWild {
			return true
		}
	}
	return false
}

// Match finds a route matching the request
func (r *Router) Match(req *http.Request) *Route {
	host := strings.ToLower(req.Host)
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	path := req.URL.Path
	method := req.Method

	for _, entry := range r.routes {
		// Check host match
		if entry.route.Host != "" && entry.route.Host != host {
			// Support wildcard host matching (*.example.com)
			if !matchWildcardHost(entry.route.Host, host) {
				continue
			}
		}

		// Check method match
		if !entry.route.Methods["*"] && !entry.route.Methods[method] {
			continue
		}

		// Check path match
		params, ok := matchPath(entry.segments, path)
		if !ok {
			continue
		}

		// Clone route with path params
		matched := *entry.route
		matched.PathParams = params
		return &matched
	}

	return nil
}

// matchWildcardHost matches patterns like *.example.com
func matchWildcardHost(pattern, host string) bool {
	if !strings.HasPrefix(pattern, "*.") {
		return false
	}
	suffix := pattern[1:] // ".example.com"
	return strings.HasSuffix(host, suffix)
}

// matchPath matches a path against segments
func matchPath(segments []segment, path string) (map[string]string, bool) {
	path = strings.Trim(path, "/")

	if len(segments) == 0 {
		return nil, path == ""
	}

	pathParts := strings.Split(path, "/")
	if path == "" {
		pathParts = nil
	}

	params := make(map[string]string)

	si := 0 // segment index
	pi := 0 // path index

	for si < len(segments) {
		seg := segments[si]

		if seg.isWild {
			// * matches one segment, ** matches rest
			if seg.value == "**" {
				// Match rest of path
				if pi < len(pathParts) {
					params["**"] = strings.Join(pathParts[pi:], "/")
				}
				return params, true
			}
			// Single wildcard
			if pi >= len(pathParts) {
				return nil, false
			}
			pi++
			si++
			continue
		}

		if seg.isParam {
			if pi >= len(pathParts) {
				return nil, false
			}
			params[seg.value] = pathParts[pi]
			pi++
			si++
			continue
		}

		// Literal match
		if pi >= len(pathParts) {
			return nil, false
		}
		if strings.ToLower(pathParts[pi]) != seg.value {
			return nil, false
		}
		pi++
		si++
	}

	// All segments matched, check if path is fully consumed
	return params, pi == len(pathParts)
}

// StripPrefix removes the matched prefix from the path
func (r *Route) StripPrefix(path string) string {
	if !r.StripPath {
		return path
	}

	// Find the static prefix to strip
	segments := parseSegments(r.Pattern)
	prefix := "/"
	for _, seg := range segments {
		if seg.isWild || seg.isParam {
			break
		}
		prefix += seg.value + "/"
	}

	prefix = strings.TrimSuffix(prefix, "/")
	if prefix == "/" {
		return path
	}

	return strings.TrimPrefix(path, prefix)
}
