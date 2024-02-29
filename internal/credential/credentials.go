// Copyright (c) 2024 The konf authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package credential

import (
	"fmt"
	"regexp"
	"unsafe"
)

func Blur(name string, value any) string {
	if namePattern.MatchString(name) {
		return "******"
	}

	var formatted string
	switch v := value.(type) {
	case string:
		formatted = v
	case []byte:
		formatted = unsafe.String(unsafe.SliceData(v), len(v))
	default:
		formatted = fmt.Sprint(value)
	}

	for name, pattern := range secretsPatterns {
		if pattern.MatchString(formatted) {
			return name
		}
	}

	return formatted
}

//nolint:gochecknoglobals,lll
var (
	namePattern     = regexp.MustCompile(`(?i)password|passwd|pass|pwd|pw|secret|token|apiKey|bearer|cred`)
	secretsPatterns = map[string]*regexp.Regexp{
		"RSA private key":                  regexp.MustCompile(`-----BEGIN RSA PRIVATE KEY-----`),
		"SSH (DSA) private key":            regexp.MustCompile(`-----BEGIN DSA PRIVATE KEY-----`),
		"SSH (EC) private key":             regexp.MustCompile(`-----BEGIN EC PRIVATE KEY-----`),
		"PGP private key block":            regexp.MustCompile(`-----BEGIN PGP PRIVATE KEY BLOCK-----`),
		"Slack Token":                      regexp.MustCompile(`xox[pborsa]-[0-9]{12}-[0-9]{12}-[0-9]{12}-[a-z0-9]{32}`),
		"AWS API Key":                      regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		"Amazon MWS Auth Token":            regexp.MustCompile(`amzn\.mws\.[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`),
		"AWS AppSync GraphQL Key":          regexp.MustCompile(`da2-[a-z0-9]{26}`),
		"GitHub personal access token":     regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),
		"GitHub fine-grained access token": regexp.MustCompile(`github_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59}`),
		"GitHub action temporary token":    regexp.MustCompile(`ghs_[a-zA-Z0-9]{36}`),
		"Google API Key":                   regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`),
		"Google Cloud Platform API Key":    regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`),
		"Google Cloud Platform OAuth":      regexp.MustCompile(`[0-9]+-[0-9A-Za-z_]{32}\.apps\.googleusercontent\.com`),
		"Google Drive API Key":             regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`),
		"Google Drive OAuth":               regexp.MustCompile(`[0-9]+-[0-9A-Za-z_]{32}\.apps\.googleusercontent\.com`),
		"Google (GCP) Service-account":     regexp.MustCompile(`"type": "service_account"`),
		"Google Gmail API Key":             regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`),
		"Google Gmail OAuth":               regexp.MustCompile(`[0-9]+-[0-9A-Za-z_]{32}\.apps\.googleusercontent\.com`),
		"Google OAuth Access Token":        regexp.MustCompile(`ya29\.[0-9A-Za-z\-_]+`),
		"Google YouTube API Key":           regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`),
		"Google YouTube OAuth":             regexp.MustCompile(`[0-9]+-[0-9A-Za-z_]{32}\.apps\.googleusercontent\.com`),
		"Generic API Key":                  regexp.MustCompile(`[aA][pP][iI]_?[kK][eE][yY].*[''|"][0-9a-zA-Z]{32,45}[''|"]`),
		"Generic Secret":                   regexp.MustCompile(`[sS][eE][cC][rR][eE][tT].*[''|"][0-9a-zA-Z]{32,45}[''|"]`),
		"Heroku API Key":                   regexp.MustCompile(`[hH][eE][rR][oO][kK][uU].*[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}`),
		"MailChimp API Key":                regexp.MustCompile(`[0-9a-f]{32}-us[0-9]{1,2}`),
		"Mailgun API Key":                  regexp.MustCompile(`key-[0-9a-zA-Z]{32}`),
		"Password in URL":                  regexp.MustCompile(`[a-zA-Z]{3,10}://[^/\\s:@]{3,20}:[^/\\s:@]{3,20}@.{1,100}["'\\s]`),
		"Slack Webhook":                    regexp.MustCompile(`https://hooks\.slack\.com/services/T[a-zA-Z0-9_]{8}/B[a-zA-Z0-9_]{8}/[a-zA-Z0-9_]{24}`),
		"Stripe API Key":                   regexp.MustCompile(`sk_live_[0-9a-zA-Z]{24}`),
		"Stripe Restricted API Key":        regexp.MustCompile(`rk_live_[0-9a-zA-Z]{24}`),
		"Square Access Token":              regexp.MustCompile(`sq0atp-[0-9A-Za-z\-_]{22}`),
		"Square OAuth Secret":              regexp.MustCompile(`sq0csp-[0-9A-Za-z\-_]{43}`),
		"Telegram Bot API Key":             regexp.MustCompile(`[0-9]+:AA[0-9A-Za-z\-_]{33}`),
		"Twilio API Key":                   regexp.MustCompile(`SK[0-9a-fA-F]{32}`),
		"Twitter Access Token":             regexp.MustCompile(`[tT][wW][iI][tT][tT][eE][rR].*[1-9][0-9]+-[0-9a-zA-Z]{40}`),
		"Twitter OAuth":                    regexp.MustCompile(`[tT][wW][iI][tT][tT][eE][rR].*[''|"][0-9a-zA-Z]{35,44}[''|"]`),
	}
)
