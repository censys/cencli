package censeye

import (
	"regexp"
)

// vendored from github.com/Censys-Research/censeye-ng/blob/main/pkg/config/defaults.go

type censeyeConfig struct {
	Filters          []string
	RgxFilters       []*regexp.Regexp
	KeyValuePrefixes []string
	ExtractionRules  []*extractionRule
}

// defaultCenseyeConfig is the default configuration for censeye analysis.
var defaultCenseyeConfig = censeyeConfig{
	Filters:          defaultFilters,
	RgxFilters:       defaultRgxFilters,
	KeyValuePrefixes: defaultKeyValuePrefixes,
	ExtractionRules:  defaultExtractionRules,
}

type extractionRule struct {
	Scope  string   `yaml:"scope,omitempty"`
	Fields []string `yaml:"fields,omitempty"`
}

// these are the default "special" rules for grouping cenql terms together.
var defaultExtractionRules = []*extractionRule{
	{
		// see services.tls.ja4s + services.protocol
		Fields: []string{"tls.ja4s", "protocol"},
	},
	{
		// see services.cert.fingerprint_sha256 + services.protocol
		Fields: []string{"cert.fingerprint_sha256", "protocol"},
	},
	{
		// see services.cert.parsed.subject_dn + services.protocol
		Fields: []string{"cert.parsed.subject_dn", "protocol"},
	},
	{
		// see services.cert.parsed.issuer_dn + services.protocol
		Fields: []string{"cert.parsed.issuer_dn", "protocol"},
	},
	{
		// get both subject and issuer locality
		Fields: []string{"cert.parsed.issuer.locality", "cert.parsed.subject.locality"},
	},
	{
		// see services.cert.parsed.subject_dn + services.cert.parsed.issuer_dn
		Fields: []string{"cert.parsed.subject_dn", "cert.parsed.issuer_dn"},
	},
	{
		// see services.endpoints.http.protocol + services.endpoints.http.status_code + services.endpoints.http.status_reason
		Scope:  "endpoints",
		Fields: []string{"http.protocol", "http.status_code", "http.status_reason"},
	},
	{
		// see services.endpoints.http.protocol + services.endpoints.http.status_code + services.endpoints.http.html_title
		Scope:  "endpoints",
		Fields: []string{"http.protocol", "http.status_code", "http.status_reason", "http.html_title"},
	},
	{
		Fields: []string{"banner_hash_sha256", "protocol"},
	},
	{
		Fields: []string{"ja4tscan.fingerprint", "banner_hash_sha256"},
	},
}

// these are things we want to filter out (by default) from the rule generation
// NOTE: anything with a trailing dot (.) is a prefix match, so we will filter out all the children.
var defaultFilters = []string{
	"host.whois.", "host.dns.",
	"host.location.",
	"host.services.threats.",
	"host.services.scan_time",
	"host.services.transport_protocol",
	"host.ip",
	"host.services.ja4tscan.scan_time",
	"host.service_count",
	"host.autonomous_system.",
	"host.services.ip",
	"host.services.vulns.",
	"host.services.endpoints.http.body",
	"host.services.banner",
	"host.services.endpoints.banner",
	"host.services.cert.parsed.validity_period.not_before",
	"host.services.cert.parsed.validity_period.not_after",
	"host.services.endpoints.scan_time",
	"host.services.tls.versions.version",
	"host.services.tls.version_selected",
	"host.services.software.evidence.literal_match",
	"host.services.software.evidence.proprietary",
	"host.services.labels.source",
	"host.services.jarm.scan_time",
	"host.services.cert.zlint.timestamp",
	"host.services.jarm.transport_protocol",
	"host.services.cert.added_at",
	"host.services.software.source",
	"host.services.cert.parse_status",
	"host.services.cert.modified_at",
	"host.services.cert.validated_at",
	"host.services.cert.parsed.subject.postal_code",
	"host.services.cert.parsed.issuer.postal_code",
	"host.services.software.",
	"host.services.endpoints.transport_protocol",
	"host.operating_system.",
	"host.services.cert.zlint.",
	"host.services.cert.ct.",
	"host.services.cert.validation.",
	"host.services.cert.revocation.",
	"host.services.labels.",
	"host.services.cert.validation_level",
	"host.services.cert.parsed.extensions.",
	"host.services.endpoints.http.body_hash_sha1", // we already have sha256...
	"host.services.cert.parsed.subject_key_info.",
	"host.services.endpoints.http.supported_versions", // don't care.
	"host.services.jarm.port",                         // internal censys thing...
	"host.services.tls.versions.",                     // dup'd info from host.services.tls.jarm
	"host.services.jarm.is_success",                   // useless boolean
	"host.services.cert.parsed.signature.valid",       // boolean
	"host.services.port",                              // no need.
	"host.services.endpoints.port",                    // no need.
	"host.services.cert.parsed.signature.",
	"host.services.cert.parent_spki_subject_fingerprint_sha256",
	"host.services.jarm.ip",
	"host.services.endpoints.hostname",
	"host.services.endpoints.endpoint_type",
	"host.services.cert.parsed.version",
	"host.services.endpoints.ip",          // stupid.
	"host.services.ssh.kex_init_message.", // not useful
	"host.services.ssh.algorithm_selection.client_to_server_alg_group.",
	"host.services.ssh.server_host_key.ecdsa_public_key.", // not useful
	"host.services.endpoints.http.uri",
	"host.services.endpoints.http.body_size",
	"host.services.operating_systems.",
	"host.services.cert.fingerprint_sha1", // dup'd by sha256
	"host.services.endpoints.http.favicons.hash_md5",
	"host.services.endpoints.http.favicons.hash_shodan",
	"host.services.ssh.algorithm_selection.server_to_client_alg_group.",
	"host.services.endpoints.http.favicons.size",
	"host.services.misconfigs.",
	"host.services.dcerpc.endpoints.",
	"host.services.endpoints.http.favicons.name",  // too specific to matter (e.g., http://1.1.1.1/favicon.ico)
	"host.services.endpoints.http.body_hash_tlsh", // basically the same thing as body_hash_sha256
	"host.services.hardware.",
	"host.services.banner_hex",                           // not useful, we have the banner hash already.
	"host.services.cert.tbs_no_ct_fingerprint_sha256",    // not useful, we have the cert fingerprint already.
	"host.services.cert.tbs_fingerprint_sha256",          // not useful, we have the cert fingerprint already.
	"host.services.cert.spki_subject_fingerprint_sha256", // not useful, we have the cert fingerprint already.
	"host.services.cert.spki_fingerprint_sha256",         // not useful, we have the cert fingerprint already.
	"host.services.cert.fingerprint_md5",                 // not useful, we have the cert fingerprint already.
	"host.services.tls.fingerprint_sha256",               // not useful, we have the cert fingerprint already.
	"host.services.cert.parsed.serial_number",            // dumb
	"host.services.cert.parsed.serial_number_hex",        // dumb
	"host.services.endpoints.open_directory.files.last_modified",
	"host.services.endpoints.open_directory.files.size",
	"host.services.endpoints.open_directory.files.extension",
	"host.services.endpoints.open_directory.files.suspicious_score",
	"host.services.endpoints.open_directory.files.path",
	"host.services.endpoints.cobalt_strike.x86.unknown_bytes.", // not useful
	"host.services.endpoints.cobalt_strike.x86.unknown_int.",   // not useful
	"host.services.endpoints.cobalt_strike.x64.unknown_int.",   // not useful
	"host.services.endpoints.cobalt_strike.x64.unknown_bytes.", // not useful
	"host.services.endpoints.cobalt_strike.x86.sleep_time",     // the sleep time can be pretty arbitrary, so not good to pivot into.
	"host.services.endpoints.cobalt_strike.x64.sleep_time",     // the sleep time can be pretty arbitrary, so not good to pivot into.
	"host.services.endpoints.cobalt_strike.x86.cookie_beacon",  // boolean value, so not that great for pivoting.
	"host.services.endpoints.cobalt_strike.x64.cookie_beacon",  // boolean value, so not that great for pivoting.
	"host.services.mysql.connection_id",                        // dumb.
	"host.services.postgres.protocol_error.",                   // doesn't work
	"host.services.postgres.startup_error.",                    // doesn't work
	"host.services.redis.raw_command_output.output",            // way too big and useless.
	"host.services.cert.parsed.validity_period.length_seconds", // meh
	"host.labels.",
	"host.services.portmap.v3_entries.",
	"host.services.dns.",                      // nope.
	"host.services.l2tp.ordered_messages_raw", // doesn't work.
	"host.services.telnet.wont.",
	"host.services.telnet.will.",
	"host.services.telnet.do.",
	"host.services.telnet.dont.",
	"host.services.smb.negotiation_log.system_time", // if we query two smb at the same time, two hosts can have the exact same time.
	"host.services.cert.names",                      // we have common_name, all the aliases usually end up being too much sometimes.
	"host.services.ntp.",
	"host.services.pptp.",
	"host.services.tls.cipher_selected",     // we already have the cipher suite, so this is redundant.
	"host.services.mqtt.connection_ack_raw", // dumb.
	"host.services.mysql.",
	"host.services.ike.",
	"host.services.ssh.algorithm_selection.", // not really useful.
	"host.services.exposures.",
	"host.services.smb.negotiation_log.",                             // not useful
	"host.services.smb.session_setup_log.",                           // not useful
	"host.services.rdp.selected_security_protocol.",                  // not useful
	"host.services.rdp.x224_cc_pdu_srcref",                           // not useful
	"host.services.rdp.version.",                                     // not useful
	"host.services.rdp.connect_response.",                            // not useful
	"host.services.smb.smb_version.major",                            // not useful
	"host.services.smb.smb_version.minor",                            // not useful
	"host.services.smb.smb_version.revision",                         // not useful
	"host.services.winrm.",                                           // nothing useful
	"host.services.cert.parsed.issuer.province",                      // not useful
	"host.services.cert.parsed.subject.province",                     // not useful
	"host.services.cert.parsed.subject.country",                      // not useful
	"host.services.cert.parsed.issuer.country",                       // not useful
	"host.services.ftp.status_code",                                  // not useful
	"host.services.ssh.server_host_key.rsa_public_key.",              // not useful
	"host.services.ssh.endpoint_id.protocol_version",                 // not useful
	"host.services.ssh.hassh_fingerprint",                            // not useful
	"host.hardware.",                                                 // not useful
	"host.services.endpoints.prometheus_target.metric_families.help", // we have the prom name, which is enough.
	"host.services.jarm.cipher_and_version_fingerprint",              // we already have the jarm fingerprint.
	"host.services.endpoints.pprof.",                                 // no.
	"host.services.smb.smb_version.version_string",                   // not useful
	"host.services.cert.parsed.ja4x",                                 // not useful, we have the ja4x fingerprint.
	"host.services.l2tp.sccrp.attribute_values.",                     // not useful
	"host.services.endpoints.cobalt_strike.x86.http_post.client",     // contains garbage unqueriable data.
	"host.services.endpoints.cobalt_strike.x86.http_get.client",      // contains garbage unqueriable data.
	"host.services.endpoints.cobalt_strike.x64.http_post.client",     // contains garbage unqueriable data.
	"host.services.endpoints.cobalt_strike.x64.http_get.client",      // contains garbage unqueriable data.
	"host.services.screenshots.",
}

var defaultRgxFilters = []*regexp.Regexp{
	regexp.MustCompile(`^host\.services\.endpoints\.http\.protocol="HTTP/1\.[01]"$`),
	regexp.MustCompile(`^host\.services\.endpoints\.path="(/|/robots\.txt)"`),
	regexp.MustCompile(`(?i)^host\.services:\(endpoints\.http\.headers\.key="date" and endpoints\.http\.headers\.value="<REDACTED>"\)$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.headers\.key="Connection" and endpoints\.http\.headers\.value="close"\)$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.status_code="(200|301|302|307|400|401|402|403|404|503)"$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.html_title="404 Not Found"$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.status_reason="Not Found"$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.protocol="HTTP/1\.[01]" and endpoints\.http\.status_code="404" and endpoints\.http\.status_reason="Not Found"\)$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.html_title="Not Found"$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.protocol="HTTP/1\.[01]" and endpoints\.http\.status_code="404" and endpoints\.http\.status_reason="Not Found" and endpoints\.http\.html_title="Not Found"\)$`),
	regexp.MustCompile(`^host\.services\.cert\.parsed\.ja4x="7022c563de38_7022c563de38_e73b053161df"$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.headers\.key="Server" and endpoints\.http\.headers\.value="Microsoft-HTTPAPI/2.0"\)$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.html_tags="<META HTTP-EQUIV=\\\"Content-Type\\\" Content=\\\"text/html; charset=us-ascii\\\">"$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.headers\.key="Content-Type" and endpoints\.http\.headers\.value="text/html; charset=us-ascii"\)$`),
	regexp.MustCompile(`^host\.services\.jarm\.tls_extensions_sha256="fd9c9d14e4f4f67f94f0359f8b28f532"$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.html_tags="<TITLE>Not Found</TITLE>"$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.body_hash_sha256="ce7127c38e30e92a021ed2bd09287713c6a923db9ffdb43f126e8965d777fbf0"$`),
	regexp.MustCompile(`^host\.services\.ja4tscan\.fingerprint="8192_2-1-3-4-8_1460_8_3-6"$`),
	regexp.MustCompile(`^host\.services\.endpoints\.banner_hash_sha256="d7de42c1e8c09cf951e3ad6248fda3ab48a60ca3eac8b25effd4b3067df8f362"$`),
	regexp.MustCompile(`^host\.services\.banner_hash_sha256="d7de42c1e8c09cf951e3ad6248fda3ab48a60ca3eac8b25effd4b3067df8f362"$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.html_title="IIS Windows Server"$`),
	regexp.MustCompile(`^host\.services\.cert\.parsed\.issuer\.organization="DigiCert Inc"$`),
	regexp.MustCompile(`^host\.services\.jarm\.fingerprint="(26d26d16d26d26d22c26d26d26d26dfd9c9d14e4f4f67f94f0359f8b28f532|14d14d16d14d14d08c14d14d14d14dfd9c9d14e4f4f67f94f0359f8b28f532)"$`),
	regexp.MustCompile(`^host\.services\.protocol="[^"]+"$`),
	regexp.MustCompile(`(?i)^host\.services:\(endpoints\.http\.headers\.key="Content-Length" and endpoints\.http\.headers\.value="[^"]+"\)$`),
	regexp.MustCompile(`cert\.parsed\.(subject_dn|issuer_dn)[:=]"C=AU, ST=Some-State, O=Internet Widgits Pty Ltd"`),
	regexp.MustCompile(`(?i)^host\.services:\(endpoints\.http\.headers\.key="Content-Type" and endpoints\.http\.headers\.value="text/plain"\)$`),
	regexp.MustCompile(`tls\.ja3s="(303951d4c50efb2e991652225a6f02b1|15af977ce25de452b96affa2addb1036|f75082535b4a79c07b31bdd0e2b7eb87|475c9302dc42b2751db9edcac3b74891)"`),
	regexp.MustCompile(`tls\.ja4s="(t120200_c02f_344b4dce5a52|t130200_1302_a56c5b993250|t120100_009d_bc98f8e001b5|t130200_1303_a56c5b993250)"`),
	regexp.MustCompile(`^host\.services\.ssh\.endpoint_id\.software_version="OpenSSH_7\.4"$`),
	regexp.MustCompile(`^host\.services\.ssh\.endpoint_id\.software_version="OpenSSH_8\.7"$`),
	regexp.MustCompile(`^host\.services\.banner_hash_sha256="be0da7ee170f9a69bc13b9e61ecfc9110c27db40f3f2e4c0ffae6741f064af8a"$`),
	regexp.MustCompile(`^host\.services\.banner_hash_sha256="e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"$`),
	regexp.MustCompile(`^host\.services\.banner_hash_sha256="45555cb663eaed691ee601ea9829a3ecb09f649e9f580f69eccc85986a831c90"$`),
	regexp.MustCompile(`^host\.services\.cert\.parsed\.(issuer|subject)\.organization="Internet Widgits Pty Ltd"$`),
	regexp.MustCompile(`(?i)^host\.services:\(endpoints\.http\.headers\.key="Content-Type" and endpoints\.http\.headers\.value="text/html"\)$`),
	regexp.MustCompile(`(?i)^host\.services:\(endpoints\.http\.headers\.key="Connection" and endpoints\.http\.headers\.value="keep-alive"\)$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.headers\.key="Server" and endpoints\.http\.headers\.value="nginx"\)$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.html_tags="<title>404 Not Found</title>"$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.protocol="HTTP/1\.[01]" and endpoints\.http\.status_code="404" and endpoints\.http\.status_reason="Not Found" and endpoints\.http\.html_title="404 Not Found"\)$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.status_reason="OK"$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.protocol="HTTP/1\.[01]" and endpoints\.http\.status_code="200" and endpoints\.http\.status_reason="OK"\)$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.headers\.key="Transfer-Encoding" and endpoints\.http\.headers\.value="chunked"\)$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.headers\.key="X-Content-Type-Options" and endpoints\.http\.headers\.value="nosniff"\)$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.headers\.key="Accept-Ranges" and endpoints\.http\.headers\.value="bytes"\)$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.headers\.key="Content-Type" and endpoints\.http\.headers\.value="text/html; charset=utf-8"\)$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.status_reason="Moved Permanently"$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.protocol="[^"]+" and endpoints\.http\.status_code="301" and endpoints\.http\.status_reason="Moved Permanently"\)$`),
	regexp.MustCompile(`(?i)^host\.services:\(endpoints\.http\.headers\.key="Content-Type" and endpoints\.http\.headers\.value="text/plain; charset=utf-8"\)$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.body_hash_sha256="b16e15764b8bc06c5c3f9f19bc8b99fa48e7894aa5a6ccdad65da49bbf564793"$`),
	regexp.MustCompile(`^host\.services\.endpoints\.banner_hash_sha256="d01d33874c1a1716026081b3b3fc90b8173203aba832874a8faf5e56e0058837"$`),
	regexp.MustCompile(`^host\.services\.banner_hash_sha256="d01d33874c1a1716026081b3b3fc90b8173203aba832874a8faf5e56e0058837"$`),
	regexp.MustCompile(`(?i)^host\.services:\(endpoints\.http\.headers\.key="Keep-Alive" and endpoints\.http\.headers\.value="timeout=5"\)$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.html_tags="<meta charset=utf-8>"$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.status_reason="NOT FOUND"$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.status_reason="Service Unavailable"$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.headers\.key="Server" and endpoints\.http\.headers\.value="Apache"\)$`),
	regexp.MustCompile(`(?i)^host\.services:\(endpoints\.http\.headers\.key="X-Frame-Options" and endpoints\.http\.headers\.value="SAMEORIGIN"\)$`),
	regexp.MustCompile(`(?i)^host\.services:\(endpoints\.http\.headers\.key="Content-Encoding" and endpoints\.http\.headers\.value="gzip"\)$`),
	regexp.MustCompile(`(?i)^host\.services:\(endpoints\.http\.headers\.key="Pragma" and endpoints\.http\.headers\.value="no-cache"\)$`),
	regexp.MustCompile(`(?i)^host\.services:\(endpoints\.http\.headers\.key="X-XSS-Protection" and endpoints\.http\.headers\.value="1; mode=block"\)$`),
	regexp.MustCompile(`(?i)^host\.services:\(endpoints\.http\.headers\.key="Vary" and endpoints\.http\.headers\.value="Accept-Encoding"\)$`),
	regexp.MustCompile(`(?i)^host\.services:\(endpoints\.http\.headers\.key="Content-Type" and endpoints\.http\.headers\.value="text/html; charset=UTF-8"\)$`),
	regexp.MustCompile(`^host\.services\.cert\.parsed\.issuer\.organization="Let's Encrypt"$`),
	regexp.MustCompile(`^host\.services\.tls\.presented_chain\.issuer_dn="C=US, O=Internet Security Research Group, CN=ISRG Root X1"$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.html_tags="<meta charset=\\\"utf-8\\\">"$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.status_reason="Found"$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.protocol="HTTP/1\.1" and endpoints\.http\.status_code="302" and endpoints\.http\.status_reason="Found"\)$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.html_title=""$`),
	regexp.MustCompile(`^host\.services\.ftp\.status_meaning="Service ready for new user\."$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.html_tags="<meta charset=\\\"UTF-8\\\">"$`),
	regexp.MustCompile(`\.(presented_chain|tls|parsed)\.(subject|issuer)_dn="C=US, O=DigiCert Inc, OU=www\.digicert\.com, CN=DigiCert Global Root (G2|CA)"$`),
	regexp.MustCompile(`^host\.services\.ssh\.endpoint_id\.raw="SSH-2\.0-OpenSSH_7\.4"$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.html_tags="<meta name=\\\"viewport\\\" content=\\\"width=device-width, initial-scale=1\.0\\\">"$`),
	regexp.MustCompile(`parsed\.issuer\.organizational_unit="www\.digicert\.com"$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.html_tags="<meta charset=\\\"utf-8\\\"/>"$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.headers\.key="Referrer-Policy" and endpoints\.http\.headers\.value="strict-origin-when-cross-origin"\)$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.headers\.key="X-Permitted-Cross-Domain-Policies" and endpoints\.http\.headers\.value="none"\)$`),
	regexp.MustCompile(`^host\.services\.endpoints\.http\.status_reason="Temporary Redirect"$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.protocol="HTTP/1\.1" and endpoints\.http\.status_code="307" and endpoints\.http\.status_reason="Temporary Redirect"\)$`),
	regexp.MustCompile(`^host\.services\.cert\.parsed\.issuer_dn="C=US, O=DigiCert Inc, OU=www\.digicert\.com, CN=RapidSSL TLS RSA CA G1"$`),
	regexp.MustCompile(`^host\.services:\(endpoints\.http\.headers\.key="X-Download-Options" and endpoints\.http\.headers\.value="noopen"\)$`),
	// don't pivot into expires headers, they are too dynamic.
	regexp.MustCompile(`(?i)^host\.services:\(endpoints\.http\.headers\.key="(Last-Modified|Expires|X-Runtime)" and endpoints\.http\.headers\.value="[^"]+"\)$`),
	// filter out open directory filenames less than 8 characters.
	regexp.MustCompile(`^host\.services\.endpoints\.open_directory\.files\.name="[^"]{1,7}\.[^"]+"$`),
	// we already have html_tags with the title, so get rid of this specific one
	regexp.MustCompile(`^host\.services\.endpoints\.http\.html_title="[^"]+"$`),
}

// DefaultKeyValuePrefixes are prefixes that should be treated as key-value objects
// where the keys and values are treated as separate fields with .key and .value suffixes
// This was generated via scripts/find_objects/main.go using fields defined here:
// https://platform.censys.io/api/fields
var defaultKeyValuePrefixes = []string{
	"web.endpoints.cobalt_strike.x86.unknown_bytes",
	"web.endpoints.cobalt_strike.x86.unknown_int",
	"web.endpoints.cobalt_strike.x64.unknown_int",
	"web.endpoints.cobalt_strike.x64.unknown_bytes",
	"host.dns.forward_dns",
	"host.services.realport.vpd",
	"host.services.endpoints.cobalt_strike.x64.unknown_int",
	"host.services.endpoints.cobalt_strike.x64.unknown_bytes",
	"host.services.endpoints.cobalt_strike.x86.unknown_bytes",
	"host.services.endpoints.cobalt_strike.x86.unknown_int",
	"host.services.telnet.do",
	"host.services.telnet.dont",
	"host.services.telnet.will",
	"host.services.telnet.wont",
	"host.services.oracle.refuse_error",
	"host.services.redis.info_response",
	"host.services.mssql.prelogin_options.unknown",
	"host.services.zeromq.subscription_match",
}
