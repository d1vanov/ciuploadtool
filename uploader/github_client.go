package uploader

import (
	"context"
	"errors"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"os"
)

type GitHubClient struct {
	client     *github.Client
	httpClient *http.Client
}

type GitHubResponse struct {
	response *github.Response
}

type GitHubRelease struct {
	release *github.RepositoryRelease
}

type GitHubReleaseAsset struct {
	asset *github.ReleaseAsset
}

func NewGitHubClient(ctx context.Context, gitHubToken string) GitHubClient {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: gitHubToken})
	tokenizedClient := oauth2.NewClient(ctx, tokenSource)
	client := github.NewClient(tokenizedClient)
	return GitHubClient{client: client, httpClient: tokenizedClient}
}

func (client GitHubClient) GetReleaseByTag(ctx context.Context, owner string, repo string, tagName string) (GitHubRelease, GitHubResponse, error) {
	if client.client == nil {
		return GitHubRelease{}, GitHubResponse{}, errors.New("GitHub client is nil")
	}
	gitHubRelease, gitHubResponse, err := client.client.Repositories.GetReleaseByTag(ctx, owner, repo, tagName)
	return GitHubRelease{release: gitHubRelease}, GitHubResponse{response: gitHubResponse}, err
}

func (client GitHubClient) CreateRelease(ctx context.Context, owner string, repo string, release Release) (GitHubRelease, GitHubResponse, error) {
	if client.client == nil {
		return GitHubRelease{}, GitHubResponse{}, errors.New("GitHub client is nil")
	}
	gitHubRelease, gitHubResponse, err := client.client.Repositories.CreateRelease(ctx, owner, repo, release.(GitHubRelease).release)
	return GitHubRelease{release: gitHubRelease}, GitHubResponse{response: gitHubResponse}, err
}

func (client GitHubClient) DeleteRelease(ctx context.Context, owner string, repo string, releaseId int) (GitHubResponse, error) {
	if client.client == nil {
		return GitHubResponse{}, errors.New("GitHub client is nil")
	}
	gitHubResponse, err := client.client.Repositories.DeleteRelease(ctx, owner, repo, releaseId)
	return GitHubResponse{response: gitHubResponse}, err
}

func (client GitHubClient) DeleteTag(ctx context.Context, owner string, repo string, tagName string) (GitHubResponse, error) {
	if client.client == nil {
		return GitHubResponse{}, errors.New("GitHub client is nil")
	}
	if client.httpClient == nil {
		return GitHubResponse{}, errors.New("GitHub http client is nil")
	}
	// GitHub guys haven't really created any actual API for tag deletion so need to do it the hard way
	deleteUrl := "https://api.github.com/repos/" + owner + "/" + repo + "/git/refs/tags/" + tagName
	request, err := http.NewRequest("DELETE", deleteUrl, nil)
	if err != nil {
		return GitHubResponse{}, err
	}

	gitHubResponse, err := client.httpClient.Do(request)
	return GitHubResponse{response: &github.Response{gitHubResponse, 0, 0, 0, 0, github.Rate{}}}, err
}

func (client GitHubClient) ListReleaseAssets(ctx context.Context, owner string, repo string, releaseId int) ([]GitHubReleaseAsset, GitHubResponse, error) {
	if client.client == nil {
		return nil, GitHubResponse{}, errors.New("GitHub client is nil")
	}
	gitHubReleaseAssets, gitHubResponse, err := client.client.Repositories.ListReleaseAssets(ctx, owner, repo, releaseId, nil)
	releaseAssets := make([]GitHubReleaseAsset, 0, len(gitHubReleaseAssets))
	for _, gitHubReleaseAsset := range gitHubReleaseAssets {
		releaseAssets = append(releaseAssets, GitHubReleaseAsset{asset: gitHubReleaseAsset})
	}
	return releaseAssets, GitHubResponse{response: gitHubResponse}, err
}

func (client GitHubClient) DeleteReleaseAsset(ctx context.Context, owner string, repo string, assetId int) (GitHubResponse, error) {
	if client.client == nil {
		return GitHubResponse{}, errors.New("GitHub client is nil")
	}
	gitHubResponse, err := client.client.Repositories.DeleteReleaseAsset(ctx, owner, repo, assetId)
	return GitHubResponse{response: gitHubResponse}, err
}

func (client GitHubClient) UploadReleaseAsset(ctx context.Context, owner string, repo string, releaseId int, assetName string,
	assetFile *os.File) (GitHubReleaseAsset, GitHubResponse, error) {
	if client.client == nil {
		return GitHubReleaseAsset{}, GitHubResponse{}, errors.New("GitHub client is nil")
	}
	var options github.UploadOptions
	options.Name = assetName
	gitHubReleaseAsset, gitHubResponse, err := client.client.Repositories.UploadReleaseAsset(ctx, owner, repo, releaseId, &options, assetFile)
	return GitHubReleaseAsset{asset: gitHubReleaseAsset}, GitHubResponse{response: gitHubResponse}, err
}

func (response GitHubResponse) GetStatusCode() int {
	if response.response == nil {
		return -1
	}
	return response.response.StatusCode
}

func (response GitHubResponse) GetStatus() string {
	if response.response == nil {
		return ""
	}
	return response.response.Status
}

func (response GitHubResponse) GetBody() io.ReadCloser {
	if response.response == nil {
		return nil
	}
	return response.response.Body
}

func (response GitHubResponse) CloseBody() {
	if response.response == nil {
		return
	}
	response.response.Body.Close()
}

func (release GitHubRelease) GetID() int {
	if release.release == nil {
		return 0
	}
	return release.release.GetID()
}

func (release GitHubRelease) GetName() string {
	if release.release == nil {
		return ""
	}
	return release.GetName()
}

func (release GitHubRelease) GetTagName() string {
	if release.release == nil {
		return ""
	}
	return release.release.GetTagName()
}

func (release GitHubRelease) GetTargetCommitish() string {
	if release.release == nil {
		return ""
	}
	return release.release.GetTargetCommitish()
}

func (release GitHubRelease) GetDraft() bool {
	if release.release == nil {
		return false
	}
	return release.release.GetDraft()
}

func (release GitHubRelease) GetPrerelease() bool {
	if release.release == nil {
		return false
	}
	return release.release.GetPrerelease()
}

func (releaseAsset GitHubReleaseAsset) GetID() int {
	if releaseAsset.asset == nil {
		return 0
	}
	return releaseAsset.asset.GetID()
}

func (releaseAsset GitHubReleaseAsset) GetName() string {
	if releaseAsset.asset == nil {
		return ""
	}
	return releaseAsset.asset.GetName()
}
