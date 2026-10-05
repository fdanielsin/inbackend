package app

import (
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strconv"
	"strings"
)

// Raisons de refus d'une origine CORS : elles décrivent le défaut sans jamais
// reprendre la valeur reçue (qui peut contenir un secret collé par erreur).
var (
	errOriginMixedWildcard = errors.New("the wildcard * cannot be combined with other origins")
	errOriginScheme        = errors.New("must start with lowercase http:// or https://")
	errOriginUserinfo      = errors.New("must not contain credentials (userinfo)")
	errOriginPath          = errors.New("must not contain a path, trailing slash, query or fragment")
	errOriginSpace         = errors.New("must not contain spaces or control characters")
	errOriginHostEmpty     = errors.New("host is missing")
	errOriginHostPattern   = errors.New("wildcard host patterns are not supported: list each origin")
	errOriginHostInvalid   = errors.New("host is not a valid lowercase ASCII host name or bracketed IP address")
	errOriginPortInvalid   = errors.New("port must be a number from 1 to 65535 without leading zero")
	errOriginPortDefault   = errors.New("default port must be omitted (80 for http, 443 for https)")
)

const (
	maxHostLength  = 253
	maxLabelLength = 63
	maxPort        = 65535
)

// ValidateAllowedOrigins vérifie chaque origine de CORS_ALLOWED_ORIGINS sans la
// corriger. Une origine est « http|https://hôte[:port] » : minuscules, sans
// chemin, slash final, userinfo, query ni fragment ; le port va de 1 à 65535, sans
// zéro de tête ni port par défaut. "*" n'est accepté que seul. L'erreur donne le
// rang (à partir de 1) de l'origine fautive dans la liste et la raison, jamais sa valeur.
func ValidateAllowedOrigins(origins []string) error {
	if i := slices.Index(origins, "*"); i >= 0 && len(origins) > 1 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS entry %d is not a valid origin: %w", i+1, errOriginMixedWildcard)
	}
	for i, origin := range origins {
		if origin == "*" {
			continue
		}
		if err := validateOrigin(origin); err != nil {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS entry %d is not a valid origin: %w", i+1, err)
		}
	}
	return nil
}

func validateOrigin(origin string) error {
	var scheme, authority string
	switch {
	case strings.HasPrefix(origin, "http://"):
		scheme, authority = "http", strings.TrimPrefix(origin, "http://")
	case strings.HasPrefix(origin, "https://"):
		scheme, authority = "https", strings.TrimPrefix(origin, "https://")
	default:
		return errOriginScheme
	}

	for _, r := range authority {
		if r <= ' ' || r == 0x7f {
			return errOriginSpace
		}
	}
	switch {
	case strings.Contains(authority, "@"):
		return errOriginUserinfo
	case strings.ContainsAny(authority, "/?#\\"):
		return errOriginPath
	}

	host, port, hasPort := splitHostPort(authority)
	if err := validateHost(host); err != nil {
		return err
	}
	if hasPort {
		return validatePort(scheme, port)
	}
	return nil
}

// splitHostPort sépare le port, qui suit le dernier « : » hors crochets IPv6.
func splitHostPort(authority string) (host, port string, hasPort bool) {
	if strings.HasPrefix(authority, "[") {
		end := strings.Index(authority, "]")
		if end == -1 || end == len(authority)-1 {
			return authority, "", false
		}
		if authority[end+1] != ':' {
			return authority, "", false
		}
		return authority[:end+1], authority[end+2:], true
	}
	host, port, hasPort = strings.Cut(authority, ":")
	return host, port, hasPort
}

func validateHost(host string) error {
	switch {
	case host == "":
		return errOriginHostEmpty
	case strings.Contains(host, "*"):
		return errOriginHostPattern
	case strings.HasPrefix(host, "["):
		return validateIPv6(host)
	case len(host) > maxHostLength:
		return errOriginHostInvalid
	}
	for label := range strings.SplitSeq(host, ".") {
		if !validLabel(label) {
			return errOriginHostInvalid
		}
	}
	return nil
}

// validateIPv6 exige la forme canonique entre crochets, celle que le navigateur envoie.
func validateIPv6(host string) error {
	if !strings.HasSuffix(host, "]") {
		return errOriginHostInvalid
	}
	inner := host[1 : len(host)-1]
	addr, err := netip.ParseAddr(inner)
	if err != nil || !addr.Is6() || addr.Zone() != "" || addr.String() != inner {
		return errOriginHostInvalid
	}
	return nil
}

func validLabel(label string) bool {
	if label == "" || len(label) > maxLabelLength || label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}
	for _, r := range label {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

func validatePort(scheme, port string) error {
	if port == "" || port[0] == '0' {
		return errOriginPortInvalid
	}
	for _, r := range port {
		if r < '0' || r > '9' {
			return errOriginPortInvalid
		}
	}
	number, err := strconv.Atoi(port)
	if err != nil || number > maxPort {
		return errOriginPortInvalid
	}
	if (scheme == "http" && number == 80) || (scheme == "https" && number == 443) {
		return errOriginPortDefault
	}
	return nil
}
