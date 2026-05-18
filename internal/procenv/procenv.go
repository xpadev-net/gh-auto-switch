package procenv

import "strings"

var TokenEnvNames = []string{"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"}

func WithGHHost(env []string, host string) []string {
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if strings.HasPrefix(e, "GH_HOST=") || isTokenEnv(e) {
			continue
		}
		out = append(out, e)
	}
	return append(out, "GH_HOST="+host)
}

func isTokenEnv(entry string) bool {
	for _, name := range TokenEnvNames {
		if strings.HasPrefix(entry, name+"=") {
			return true
		}
	}
	return false
}
