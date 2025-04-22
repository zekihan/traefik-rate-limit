package utils

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

var (
	Version = "dev"

	CommitHash = "none"

	BuildDate       = "I don't remember exactly"
	buildDateString = "I don't remember exactly"
	buildDateOnce   sync.Once

	GitTreeState = "not a git repo"
)

func GetBuildDate() string {
	buildDateOnce.Do(func() {
		a, err := strconv.ParseInt(BuildDate, 10, 64)
		if err != nil {
			buildDateString = BuildDate
			return
		}
		buildDateString = time.Unix(a, 0).UTC().Format(time.DateTime)
	})
	return buildDateString
}

func GetGitTreeStatePostfix() string {
	dirty := ""
	if GitTreeState == "dirty" {
		dirty = " (dirty)"
	}
	return dirty
}

func GetStartupInfo() string {
	return fmt.Sprintf("Version: %s, Build Date: %s, CommitHash: %s%s", Version, GetBuildDate(), CommitHash, GetGitTreeStatePostfix())
}
