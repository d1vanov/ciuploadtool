package uploader

import (
	"github.com/xanzy/go-gitlab"
)

type GitLabClient struct {
	client *gitlab.Client
	owner  string
	repo   string
}

type GitLabResponse struct {
	response *gitlab.Response
}

type GitLabRelease struct {
	release *gitlab.Release
}

type GitLabReleaseAsset struct {
}

func newGitLabClient(gitLabToken string, owner string, repo string) Client {
	client := gitlab.NewClient(nil, gitLabToken)
	return GitLabClient{client: client, owner: owner, repo: repo}
}
