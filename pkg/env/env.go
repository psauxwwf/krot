package env

import "os"

func IsGitHubActions() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}
