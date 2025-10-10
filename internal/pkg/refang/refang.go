package refang

import (
	"regexp"
	"strings"
)

// Precompiled patterns for performance and clarity
var (
	// IPv4 defang patterns: [.], (.), \. and partials with optional spaces
	reIPv4 = regexp.MustCompile(`\[\s*\.\s*\]|\(\s*\.\s*\)|\\\.[\)\]]?|\[\s*\.|\.\s*[\]\)]|\(\s*\.|\.\s*[\)]`)

	// IPv6 defang patterns: [:], (:), \: and partials with optional spaces
	reIPv6 = regexp.MustCompile(`\[\s*:\s*\]|\(\s*:\s*\)|\\:[\)\]]?|\[\s*:|:\s*[\]\)]|\(\s*:|:\s*[\)]`)

	// URL heuristics
	reHasProtocol            = regexp.MustCompile(`(?i)(hxxp|https?|ftp)[:\[_\\]`)
	reHasDefang              = regexp.MustCompile(`\[\.\]|\(\.\)|\\\.|\[/\]|\(\.|\\:|\[\:\]|\(\:\)`)
	reHasURLEnc              = regexp.MustCompile(`%[0-9A-Fa-f]{2}`)
	reHasSubdomain           = regexp.MustCompile(`^[A-Za-z0-9][-A-Za-z0-9]*\.[A-Za-z0-9][-A-Za-z0-9]*\.[A-Za-z]{2,63}`)
	reHasGenericTLD          = regexp.MustCompile(`^[A-Za-z0-9][-A-Za-z0-9]*\.[A-Za-z]{2,63}(/|:|$)`)
	reHasGenericTLDWithDelim = regexp.MustCompile(`^[A-Za-z0-9][-A-Za-z0-9]*\.[A-Za-z]{2,63}(/|:)`)
	// Curated common TLD list for conservative scheme addition
	reHasCommonTLD = regexp.MustCompile(`^[A-Za-z0-9][-A-Za-z0-9]*\.(com|org|net|edu|gov|mil|co|io|dev|app|ai|info|biz|me|us|uk|ca|de|fr|jp|cn|au|br|in|ru|nl|se|no|dk|es|it|ch|be|at|nz|za|mx|ar|cl|pl|tr|kr|sg|hk|tw|th|my|id|ph|vn|ae|sa|eg|ng|ke|ma|tz|gh|zm|ug|zw|mw|ao|cd|cm|ci|sn|ml|bf|bj|ne|tg|gn|lr|sl|gm|gw|mr|cv|st|ga|cg|td|cf|sd|ss|er|dj|so|et|bi|rw|ly|tn|dz|eh|yt|re|mu|sc|km|mg|mz|bw|ls|sz|na|za)(/|:|$)`)
	reIsIPv4       = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	reIsIPv6Like   = regexp.MustCompile(`^\[?[0-9A-Fa-f]{0,4}(?::[0-9A-Fa-f]{0,4}){2,}\]?`)

	// Misc helpers
	reScheme          = regexp.MustCompile(`(?i)^[a-z][a-z0-9+.-]*://`)
	reProtoSepSpaces  = regexp.MustCompile(`:\s*//\s*`)
	reSpacesDot       = regexp.MustCompile(`\s+\.\s+`)
	reSpacesSlash     = regexp.MustCompile(`\s+/\s+`)
	reProtoHostSpace  = regexp.MustCompile(`(://)\s+`)
	reLeadingSlashSp  = regexp.MustCompile(`\s+/`)
	reBracketSlash    = regexp.MustCompile(`\[/\]`)
	rePct2E           = regexp.MustCompile(`%2[Ee]`)
	rePct2F           = regexp.MustCompile(`%2[Ff]`)
	rePct3A           = regexp.MustCompile(`%3[Aa]`)
	reProtocolAndRest = regexp.MustCompile(`^([a-z][a-z0-9+.-]*://)(.*)`)
	reFirstSlash      = regexp.MustCompile(`/`)
	reSpacesAnywhere  = regexp.MustCompile(`\s+`)
)

// RefangIP refangs an IP address.
// If there is no defanging detected, the original string is returned.
// This does not perform any validation of the IP address.
// Reference: https://github.com/InQuest/iocextract?tab=readme-ov-file#more-details
//
//nolint:revive
func RefangIP(s string) string {
	res := s
	if v4 := refangIPv4(s); v4 != s {
		res = v4
	} else if v6 := refangIPv6(s); v6 != s {
		res = v6
	}
	return res
}

// refangIPv4 refangs an IPv4 address.
// Supports: [.], (.), \., and any combination/partial patterns
// Also handles space variations like "( ." or ". )"
func refangIPv4(s string) string { return reIPv4.ReplaceAllString(s, ".") }

// refangIPv6 refangs an IPv6 address.
// Supports: [:], (:), \:, and any combination/partial patterns
// Special handling: don't mangle properly formatted IPv6 addresses in brackets like [::1]:port or [2001::]:port
func refangIPv6(s string) string {
	// Check if this looks like a properly formatted IPv6 in brackets followed by :port
	// Pattern: starts with [ followed by hex chars/colons, has ]:digit
	if strings.HasPrefix(s, "[") && strings.Contains(s, "]:") {
		// Check if what's between [ and ]: looks like a valid IPv6 (contains :: or multiple colons)
		bracketEnd := strings.Index(s, "]:")
		if bracketEnd > 0 {
			ipPart := s[1:bracketEnd]
			// If it contains defang patterns like [:]  or (:), it needs refanging
			if strings.Contains(ipPart, "[:]") || strings.Contains(ipPart, "(:") || strings.Contains(ipPart, ":]") {
				// Contains defang patterns, proceed with refanging
				return reIPv6.ReplaceAllString(s, ":")
			}
			// If it has :: or multiple colons and no defang patterns, it's likely a real IPv6
			if strings.Contains(ipPart, "::") || strings.Count(ipPart, ":") >= 2 {
				// This looks like a properly formatted IPv6 with port, don't refang it
				return s
			}
		}
	}
	return reIPv6.ReplaceAllString(s, ":")
}

// RefangURL refangs a URL.
// If there is no defanging detected, the original string is returned.
// This does not perform any validation of the URL.
// Reference: https://github.com/InQuest/iocextract?tab=readme-ov-file#more-details
//
//nolint:revive
func RefangURL(s string) string {
	result := s

	// Use a trimmed version for URL-likeness detection only
	trimmed := strings.TrimSpace(result)
	hasProtocol := reHasProtocol.MatchString(trimmed)
	hasDefang := reHasDefang.MatchString(trimmed)
	hasURLEncoding := reHasURLEnc.MatchString(trimmed)
	hasSubdomain := reHasSubdomain.MatchString(trimmed)
	hasGenericTLD := reHasGenericTLD.MatchString(trimmed)
	isIPv4 := reIsIPv4.MatchString(trimmed)
	isIPv6 := reIsIPv6Like.MatchString(trimmed)

	// Only process if it looks like a URL or recognizable host
	if !hasProtocol && !hasDefang && !hasURLEncoding && !hasSubdomain && !hasGenericTLD && !isIPv4 && !isIPv6 {
		return result
	}

	// Replace hxxp/hXXp with http
	result = regexp.MustCompile(`(?i)hxxp`).ReplaceAllString(result, "http")

	// Replace simple defang variants using a fast replacer
	result = strings.NewReplacer(
		"__", "://",
		":\\\\", "://",
	).Replace(result)

	// Replace [:] with :
	result = regexp.MustCompile(`\[\:\]`).ReplaceAllString(result, ":")

	// Remove Cisco ESA spaces (spaces around protocol, dots, and slashes)
	result = reProtoSepSpaces.ReplaceAllString(result, "://")
	result = reSpacesDot.ReplaceAllString(result, ".")
	result = reSpacesSlash.ReplaceAllString(result, "/")

	// Remove spaces between protocol and hostname
	result = reProtoHostSpace.ReplaceAllString(result, "$1")
	// Remove spaces before slashes
	result = reLeadingSlashSp.ReplaceAllString(result, "/")

	// Replace [/] with /
	result = reBracketSlash.ReplaceAllString(result, "/")

	// URL decode %2E, %2F, %3A
	result = rePct2E.ReplaceAllString(result, ".")
	result = rePct2F.ReplaceAllString(result, "/")
	result = rePct3A.ReplaceAllString(result, ":")

	// Refang IPv4 and IPv6 patterns in the URL (also handles parentheses)
	result = refangIPv4(result)
	result = refangIPv6(result)

	// Remove any remaining spaces in hostname (before first / or end)
	if reScheme.MatchString(result) {
		// Has protocol - remove spaces in hostname part
		parts := reProtocolAndRest.FindStringSubmatch(result)
		if len(parts) == 3 {
			protocol := parts[1]
			rest := parts[2]
			// Find first / to separate hostname from path
			if slashIdx := reFirstSlash.FindStringIndex(rest); slashIdx != nil {
				hostname := rest[:slashIdx[0]]
				path := rest[slashIdx[0]:]
				hostname = reSpacesAnywhere.ReplaceAllString(hostname, "")
				result = protocol + hostname + path
			} else {
				// No path, just hostname
				rest = reSpacesAnywhere.ReplaceAllString(rest, "")
				result = protocol + rest
			}
		}
	}

	// Add http:// prefix if missing after refanging
	if !reScheme.MatchString(result) {
		// Re-check patterns after refanging since patterns may have changed
		trimmedAfter := strings.TrimSpace(result)
		hasSubdomainAfter := reHasSubdomain.MatchString(trimmedAfter)
		hasGenericTLDAfter := reHasGenericTLDWithDelim.MatchString(trimmedAfter)
		isIPv4After := reIsIPv4.MatchString(trimmedAfter)
		isIPv6After := reIsIPv6Like.MatchString(trimmedAfter)

		if hasSubdomainAfter || reHasCommonTLD.MatchString(trimmedAfter) || hasGenericTLDAfter || isIPv4After || isIPv6After {
			result = "http://" + result
		}
	}

	// After ensuring scheme may have been added, remove spaces in hostname again
	if reScheme.MatchString(result) {
		parts := reProtocolAndRest.FindStringSubmatch(result)
		if len(parts) == 3 {
			protocol := parts[1]
			rest := parts[2]
			if slashIdx := reFirstSlash.FindStringIndex(rest); slashIdx != nil {
				hostname := rest[:slashIdx[0]]
				path := rest[slashIdx[0]:]
				hostname = reSpacesAnywhere.ReplaceAllString(hostname, "")
				result = protocol + hostname + path
			} else {
				rest = reSpacesAnywhere.ReplaceAllString(rest, "")
				result = protocol + rest
			}
		}
	}

	// Final trim to remove leading/trailing whitespace introduced by inputs
	result = strings.TrimSpace(result)

	return result
}
