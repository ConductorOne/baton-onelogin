package connector

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
)

var ResourcesPageSize = 50

func annotationsForUserResourceType() annotations.Annotations {
	annos := annotations.Annotations{}
	annos.Update(&v2.SkipEntitlementsAndGrants{})
	return annos
}

func parsePageToken(i string, resourceID *v2.ResourceId) (*pagination.Bag, string, error) {
	b := &pagination.Bag{}
	err := b.Unmarshal(i)
	if err != nil {
		return nil, "", err
	}

	if b.Current() == nil {
		b.Push(pagination.PageState{
			ResourceTypeID: resourceID.ResourceType,
			ResourceID:     resourceID.Resource,
		})
	}

	return b, b.PageToken(), nil
}

var disallowedSchemesAsHostnames = map[string]bool{
	"javascript": true,
	"mailto":     true,
	"ftp":        true,
	"file":       true,
}

const suffix = "onelogin.com"

func sanitizeDomainInput(domain string) (string, error) {
	var host string
	var port string

	if strings.Contains(domain, "//") {
		u, err := url.Parse(domain)
		if err != nil {
			return "", fmt.Errorf("invalid URL format: %w", err)
		}
		scheme := strings.ToLower(u.Scheme)

		if scheme != "http" && scheme != "https" && scheme != "htp" && scheme != "htps" {
			return "", fmt.Errorf("unsupported URL scheme: '%s'", u.Scheme)
		}
		host = u.Hostname()
		port = u.Port()
		if host == "" {
			return "", fmt.Errorf("could not extract hostname from URL: '%s'", domain)
		}
	} else {
		// No "://" found, try to parse as host or host:port
		tempHost, tempPort, err := net.SplitHostPort(domain)
		if err == nil {
			// Successfully split host and port
			// Check if the extracted 'host' part looks like a disallowed scheme
			if disallowedSchemesAsHostnames[strings.ToLower(tempHost)] {
				return "", fmt.Errorf("invalid hostname part, looks like a scheme: '%s'", tempHost)
			}
			host = tempHost
			port = tempPort
		} else {
			// Could not split host:port, assume the whole input is the host
			// But first, check if it contains a colon, which might indicate an invalid scheme attempt without ://
			if strings.Contains(domain, ":") {
				firstPart := strings.SplitN(domain, ":", 2)[0]
				if disallowedSchemesAsHostnames[strings.ToLower(firstPart)] {
					return "", fmt.Errorf("invalid input, looks like a scheme without '://': '%s'", firstPart)
				}
			}
			host = domain
		}
	}

	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("hostname could not be determined from input: '%s'", domain)
	}

	host = strings.ToLower(host)
	parts := strings.Split(host, ".")

	didAppendSuffix := false
	// Handle single component expansion (e.g., "acme" -> "acme.okta.com")
	if len(parts) == 1 && parts[0] != "" {
		host = strings.Join(append(parts, suffix), ".") // Update host for suffix check
		didAppendSuffix = true
	}

	// Suffix validation

	matched := false

	if host == suffix {
		if didAppendSuffix { // Case: single input component + appendSuffix resulted in exact match
			matched = true
		} else if len(parts) > strings.Count(suffix, ".")+1 {
			// Case: multi-component input (or no appendSuffix) was an exact match for suffix
			// 'parts' here refers to the components of the original host before any appendSuffix.
			matched = true
		}
	} else if strings.HasSuffix(host, "."+suffix) {
		// Check that there's something before the suffix
		if len(strings.TrimSuffix(host, "."+suffix)) > 0 {
			matched = true
		}
	}

	if !matched {
		return "", fmt.Errorf("domain '%s' does not have an expected suffix (e.g., %s)", host, suffix)
	}

	// Re-split parts after potential single-component expansion for remapping/validation
	parts = strings.Split(host, ".")

	normalizedHost := strings.Join(parts, ".")

	if port != "" && port != "80" && port != "443" {
		normalizedHost = net.JoinHostPort(normalizedHost, port)
	}

	subdomain := strings.TrimSuffix(host, ".onelogin.com")

	return subdomain, nil
}
