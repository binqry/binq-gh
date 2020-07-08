package cli

import (
	"context"
	"os"

	"github.com/google/go-github/v32/github"
	"github.com/progrhyme/go-lv"
	"golang.org/x/oauth2"
)

func newGitHubClient(ctx context.Context, token string) (client *github.Client) {
	if token == "" {
		token = os.Getenv(envGitHubTokenKey)
	}
	if token == "" {
		lv.Noticef("GitHub API Token not set. Recommend to set it by $GITHUB_TOKEN or -t|--token option")
		return github.NewClient(nil)
	}

	lv.Debugf("GitHub API Token is set")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}
