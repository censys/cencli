package refang

import "testing"

func TestRefangIP(t *testing.T) {
	tests := []struct {
		name, input, expected string
	}{
		// IPv4: Basic defang patterns
		{"ipv4 square brackets", "1[.]1[.]1[.]1", "1.1.1.1"},
		{"ipv4 parentheses", "1(.)1(.)1(.)1", "1.1.1.1"},
		{"ipv4 backslash escape", "1\\.1\\.1\\.1", "1.1.1.1"},
		{"ipv4 partial defang mixed", "1[.1[.1.]1", "1.1.1.1"},
		{"ipv4 mixed backslash and brackets", "1\\.)1[.1.)1", "1.1.1.1"},
		{"ipv4 mixed all three patterns", "192[.]168\\.1(.)1", "192.168.1.1"},
		{"ipv4 mixed order 1", "10(.)0[.]0\\.1", "10.0.0.1"},
		{"ipv4 mixed order 2", "8\\.8(.)8[.]8", "8.8.8.8"},
		{"ipv4 google dns brackets", "8[.]8[.]8[.]8", "8.8.8.8"},
		{"ipv4 cloudflare dns parens", "1(.)1(.)1(.)1", "1.1.1.1"},
		{"ipv4 private network", "192[.]168[.]1[.]254", "192.168.1.254"},
		{"ipv4 localhost", "127[.]0[.]0[.]1", "127.0.0.1"},
		{"ipv4 normal unchanged", "1.1.1.1", "1.1.1.1"},
		{"ipv4 normal different octets", "192.168.1.1", "192.168.1.1"},
		{"ipv4 normal max octets", "255.255.255.255", "255.255.255.255"},
		{"ipv4 single octet defanged", "1[.]2", "1.2"},
		{"ipv4 five octets", "10[.]20[.]30[.]40[.]50", "10.20.30.40.50"},
		{"ipv4 invalid large numbers", "999[.]999[.]999[.]999", "999.999.999.999"},
		{"ipv4 with port", "192[.]168[.]1[.]1:8080", "192.168.1.1:8080"},
		{"ipv4 in url", "http://192[.]168[.]1[.]1/path", "http://192.168.1.1/path"},
		{"ipv4 only opening bracket", "1[.2.3.4", "1.2.3.4"},
		{"ipv4 only opening paren", "1(.)2.3.4", "1.2.3.4"},
		{"ipv4 mix with normal dots", "1[.]2.3[.]4", "1.2.3.4"},
		{"ipv4 multiple in text", "Connect from 1[.]1[.]1[.]1 to 8[.]8[.]8[.]8", "Connect from 1.1.1.1 to 8.8.8.8"},
		{"ipv4 multiple mixed patterns", "Source: 10(.)0(.)0(.)1, Dest: 192[.]168\\.1\\.254", "Source: 10.0.0.1, Dest: 192.168.1.254"},

		// IPv6: Basic defang patterns
		{"ipv6 square brackets", "2001[:]db8[:]85a3[:]0[:]0[:]8a2e[:]370[:]7334", "2001:db8:85a3:0:0:8a2e:370:7334"},
		{"ipv6 parentheses", "2001(:)db8(:)85a3(:)0(:)0(:)8a2e(:)370(:)7334", "2001:db8:85a3:0:0:8a2e:370:7334"},
		{"ipv6 backslash", "2001\\:db8\\:85a3\\:0\\:0\\:8a2e\\:370\\:7334", "2001:db8:85a3:0:0:8a2e:370:7334"},
		{"ipv6 mixed patterns", "2001[:]db8(:)85a3\\:0[:]0(:)8a2e\\:370[:]7334", "2001:db8:85a3:0:0:8a2e:370:7334"},
		{"ipv6 compressed brackets", "2001[:]db8[:][:]1", "2001:db8::1"},
		{"ipv6 compressed parens", "fe80(:)(:)1", "fe80::1"},
		{"ipv6 compressed backslash", "[::]1", "::1"},
		{"ipv6 loopback brackets", "[:][:]1", "::1"},
		{"ipv6 loopback parens", "(:)(:)1", "::1"},
		{"ipv6 normal unchanged", "2001:db8::1", "2001:db8::1"},
		{"ipv6 full address", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", "2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
		{"ipv6 partial defang", "2001[:db8[::]1", "2001:db8::1"},
		{"ipv6 with port", "[2001[:]db8[:][:]1]:8080", "[2001:db8::1]:8080"},
		{"ipv6 in url", "http://[2001[:]db8[:][:]1]/path", "http://[2001:db8::1]/path"},

		// Edge cases
		{"empty string", "", ""},
		{"only dots", "...", "..."},
		{"defanged dots no numbers", "[.][.][.]", "..."},
		{"only colons", ":::", ":::"},
		{"defanged colons no numbers", "[:][:][:][:]", "::::"},
		{"plain text", "hello world", "hello world"},
		{"brackets without dots/colons", "test[123]", "test[123]"},
		{"parens without dots/colons", "func(test)", "func(test)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := RefangIP(tt.input); result != tt.expected {
				t.Errorf("RefangIP(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRefangURL(t *testing.T) {
	tests := []struct {
		name, input, expected string
	}{
		// Basic dot defang patterns
		{"dot square brackets", "example[.]com/path", "http://example.com/path"},
		{"dot parentheses", "example(.)com/path", "http://example.com/path"},
		{"dot backslash", "example\\.com/path", "http://example.com/path"},
		{"partial dot brackets", "http://example[.com/path", "http://example.com/path"},
		{"multiple dot brackets", "sub[.]example[.]com/path", "http://sub.example.com/path"},

		// Slash defang patterns
		{"slash brackets", "http://example.com[/]path", "http://example.com/path"},
		{"multiple slash brackets", "http://example.com[/]path[/]to[/]file", "http://example.com/path/to/file"},

		// Cisco ESA spaces
		{"cisco esa spaces", "http:// example .com /path", "http://example.com/path"},
		{"cisco esa multiple spaces", "http:// sub . example . com / path", "http://sub.example.com/path"},

		// Protocol defang patterns
		{"double underscore", "http__example.com/path", "http://example.com/path"},
		{"backslash colon", "http:\\\\example.com/path", "http://example.com/path"},
		{"colon brackets", "http[:]//example.com/path", "http://example.com/path"},
		{"hxxp lowercase", "hxxp://example.com/path", "http://example.com/path"},
		{"hxxp uppercase", "HXXP://example.com/path", "http://example.com/path"},
		{"hxxp mixed case", "hXxP://example.com/path", "http://example.com/path"},

		// URL encoding
		{"url encoded dots", "http%3A%2F%2fexample%2Ecom%2Fpath", "http://example.com/path"},
		{"url encoded mixed", "http%3a%2f%2fexample%2ecom%2fpath", "http://example.com/path"},
		{"partial url encoding", "http://example%2Ecom/path", "http://example.com/path"},

		// Combination patterns
		{"combo all patterns", "hxxp__ example( .com[/]path", "http://example.com/path"},
		{"combo hxxp and brackets", "hxxp://example[.]com/path", "http://example.com/path"},
		{"combo backslash and cisco", "http:\\\\ example .com /path", "http://example.com/path"},
		{"combo mixed defang", "hxxp[:]//example[.]com[/]api[/]v1", "http://example.com/api/v1"},

		// With query strings and fragments
		{"with query string", "example[.]com/path?query=value", "http://example.com/path?query=value"},
		{"with fragment", "example[.]com/path#section", "http://example.com/path#section"},
		{"with query and fragment", "example[.]com/path?q=v#sec", "http://example.com/path?q=v#sec"},

		// Different protocols
		{"https protocol", "https://example[.]com/path", "https://example.com/path"},
		{"ftp protocol", "ftp://example[.]com/file", "ftp://example.com/file"},
		{"hxxps protocol", "hxxps://example[.]com/path", "https://example.com/path"},

		// With ports
		{"with port defanged", "http://example[.]com:8080/path", "http://example.com:8080/path"},
		{"with port and brackets", "http[:]//example[.]com:8080/path", "http://example.com:8080/path"},

		// IPv4 in URLs
		{"ipv4 defanged brackets", "http://192[.]168[.]1[.]1/path", "http://192.168.1.1/path"},
		{"ipv4 defanged mixed", "http://10(.)0(.)0\\.1/path", "http://10.0.0.1/path"},
		{"ipv4 no protocol", "192[.]168[.]1[.]1/path", "http://192.168.1.1/path"},

		// Edge cases
		{"already normal", "http://example.com/path", "http://example.com/path"},
		{"no protocol normal", "example.com/path", "http://example.com/path"},
		{"subdomain normal", "sub.example.com/path", "http://sub.example.com/path"},
		{"no path", "example[.]com", "http://example.com"},
		{"trailing slash", "example[.]com/", "http://example.com/"},
		{"deep path", "example[.]com/a/b/c/d", "http://example.com/a/b/c/d"},

		// Should not modify
		{"plain text", "this is plain text", "this is plain text"},
		{"email address", "user@example.com", "user@example.com"},
		{"no domain pattern", "just[.]text", "just.text"},

		// Complex real-world examples
		{"malware url", "hxxp://malicious[.]example[.]com/payload[.]exe", "http://malicious.example.com/payload.exe"},
		{"phishing url", "hxxps[:]//secure[.]bank[.]phishing[.]com/login", "https://secure.bank.phishing.com/login"},
		{"c2 callback", "http__ 10(.)20(.)30(.)40 /callback", "http://10.20.30.40/callback"},
		{"ipv6 bracket with spaces", "http://[2001[:] db8 [:] [:] 1]/p", "http://[2001:db8::1]/p"},
		{"ipv6 paren with spaces", "hxxp://fe80( :) ( :) 1/path", "http://fe80::1/path"},
		{"ipv6 backslash url no scheme", "[2001\\:db8\\:85a3\\:0\\:0\\:8a2e\\:370\\:7334]/x", "http://[2001:db8:85a3:0:0:8a2e:370:7334]/x"},
		{"generic tld long", "example.technology/path", "http://example.technology/path"},
		{"subdomain generic tld", "sub.example.agency/p", "http://sub.example.agency/p"},
		{"trim gating leading spaces", "   example[.]net/x", "http://example.net/x"},
		{"trim gating trailing spaces", "example[.]io/x   ", "http://example.io/x"},
		{"ipv6 no scheme host only", "[2001[:]db8[:][:]1]", "http://[2001:db8::1]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := RefangURL(tt.input); result != tt.expected {
				t.Errorf("RefangURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
