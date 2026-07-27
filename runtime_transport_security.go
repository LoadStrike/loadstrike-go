package loadstrike

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type runtimeTransportPolicy struct {
	scheme            string
	host              string
	allowLoopbackHTTP bool
}

func newRuntimeTransportPolicy(
	allowedOrigin string,
	allowLoopbackHTTP bool,
) (runtimeTransportPolicy, error) {
	origin := allowedOrigin
	if candidate, err := url.Parse(allowedOrigin); err == nil && candidate.Path == "" {
		origin += "/"
	}
	parsed, err := parseCanonicalRuntimeURL(origin)
	if err != nil {
		return runtimeTransportPolicy{}, fmt.Errorf(
			"runtime artifact origin is invalid: %w",
			err,
		)
	}
	switch parsed.Scheme {
	case "https":
	case "http":
		if !allowLoopbackHTTP || !isLoopbackRuntimeHost(parsed.Hostname()) {
			return runtimeTransportPolicy{}, errors.New(
				"runtime artifact origin must use HTTPS",
			)
		}
	default:
		return runtimeTransportPolicy{}, errors.New(
			"runtime artifact origin must use HTTPS",
		)
	}
	return runtimeTransportPolicy{
		scheme:            parsed.Scheme,
		host:              strings.ToLower(parsed.Host),
		allowLoopbackHTTP: allowLoopbackHTTP,
	}, nil
}

func (policy runtimeTransportPolicy) validateResolveURL(raw string) (*url.URL, error) {
	return policy.validateURL(raw, "runtime resolve URL")
}

func (policy runtimeTransportPolicy) validateArtifactURL(raw string) (*url.URL, error) {
	return policy.validateURL(raw, "runtime artifact URL")
}

func (policy runtimeTransportPolicy) validateURL(
	raw string,
	label string,
) (*url.URL, error) {
	parsed, err := parseCanonicalRuntimeURL(raw)
	if err != nil {
		return nil, fmt.Errorf("%s is invalid: %w", label, err)
	}
	if parsed.Scheme != policy.scheme ||
		strings.ToLower(parsed.Host) != policy.host {
		return nil, fmt.Errorf("%s origin is not allowed", label)
	}
	if parsed.Scheme != "https" &&
		(!policy.allowLoopbackHTTP || !isLoopbackRuntimeHost(parsed.Hostname())) {
		return nil, fmt.Errorf("%s must use HTTPS", label)
	}
	return parsed, nil
}

func parseCanonicalRuntimeURL(raw string) (*url.URL, error) {
	if raw == "" || raw != strings.TrimSpace(raw) {
		return nil, errors.New("URL must be non-empty without surrounding whitespace")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, errors.New("URL syntax is invalid")
	}
	if !parsed.IsAbs() ||
		parsed.Opaque != "" ||
		parsed.Host == "" ||
		parsed.User != nil ||
		parsed.RawQuery != "" ||
		parsed.ForceQuery ||
		parsed.Fragment != "" ||
		parsed.RawFragment != "" ||
		parsed.RawPath != "" {
		return nil, errors.New(
			"URL must be absolute and canonical without credentials, query, fragment, or encoded path ambiguity",
		)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, errors.New("URL scheme must be HTTPS or allowed loopback HTTP")
	}
	if parsed.Path == "" ||
		!strings.HasPrefix(parsed.Path, "/") ||
		strings.Contains(parsed.Path, "\\") ||
		strings.Contains(parsed.Path, "//") ||
		path.Clean(parsed.Path) != parsed.Path {
		return nil, errors.New("URL path must be absolute and canonical")
	}
	return parsed, nil
}

func isLoopbackRuntimeHost(host string) bool {
	normalized := strings.ToLower(strings.TrimSpace(host))
	return normalized == "localhost" ||
		normalized == "127.0.0.1" ||
		normalized == "::1" ||
		strings.HasSuffix(normalized, ".localhost")
}

func rejectRuntimeArtifactRedirect(
	*http.Request,
	[]*http.Request,
) error {
	return errors.New("runtime artifact redirects are not allowed")
}

func validateRuntimeResponseURL(
	response *http.Response,
	expected *url.URL,
	policy runtimeTransportPolicy,
	label string,
) error {
	if response == nil || response.Request == nil || response.Request.URL == nil {
		return fmt.Errorf("%s did not provide a final response URL", label)
	}
	actual, err := policy.validateURL(response.Request.URL.String(), label+" response URL")
	if err != nil {
		return err
	}
	if actual.String() != expected.String() {
		return fmt.Errorf("%s final response URL did not match the request", label)
	}
	return nil
}
