package cli

import (
	"github.com/google/go-github/v32/github"
)

type nestedMapOfReleaseAsset map[string]map[string]*github.ReleaseAsset

func (nm *nestedMapOfReleaseAsset) set(name, kind string, asset *github.ReleaseAsset) {
	if (*nm)[name] == nil {
		(*nm)[name] = make(map[string]*github.ReleaseAsset)
	}
	(*nm)[name][kind] = asset
}
