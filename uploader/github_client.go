package uploader

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"os"
)

type GitHubClient struct {
	client     *github.Client
	httpClient *http.Client
	ctx        context.Context
	owner      string
	repo       string
}

type GitHubResponse struct {
	response *github.Response
}

type GitHubRelease struct {
	release *github.RepositoryRelease
	repoTag *github.RepositoryTag
}

type GitHubReleaseAsset struct {
	asset *github.ReleaseAsset
}

func newGitHubClient(gitHubToken string, owner string, repo string) Client {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: gitHubToken})
	ctx := context.Background()
	tokenizedClient := oauth2.NewClient(ctx, tokenSource)
	client := github.NewClient(tokenizedClient)
	return GitHubClient{client: client, httpClient: tokenizedClient, ctx: ctx, owner: owner, repo: repo}
}

func newGitHubRelease(releaseBody string, info *buildEventInfo) Release {
	release := GitHubRelease{release: new(github.RepositoryRelease), repoTag: new(github.RepositoryTag)}
	release.release.TagName = new(string)
	*release.release.TagName = info.tag
	release.release.TargetCommitish = new(string)
	*release.release.TargetCommitish = info.commit
	release.release.Name = new(string)
	*release.release.Name = info.releaseTitle
	release.release.Body = new(string)
	*release.release.Body = releaseBody
	release.release.Prerelease = new(bool)
	*release.release.Prerelease = info.isPrerelease
	return updateBuildLogWithinReleaseBody(release, info)
}

func (client GitHubClient) GetContext() context.Context {
	return client.ctx
}

func (client GitHubClient) GetOwner() string {
	return client.owner
}

func (client GitHubClient) GetRepo() string {
	return client.repo
}

func (client GitHubClient) GetReleaseByTag(tagName string) (Release, Response, error) {
	if client.client == nil {
		return GitHubRelease{}, GitHubResponse{}, errors.New("GitHub client is nil")
	}
	gitHubRelease, gitHubResponse, err := client.client.Repositories.GetReleaseByTag(client.ctx, client.owner, client.repo, tagName)
	if err != nil {
		return GitHubRelease{}, GitHubResponse{}, err
	}

	tagList, gitHubTagListResponse, err := client.client.Repositories.ListTags(client.ctx, client.owner, client.repo, nil)
	if err != nil {
		return GitHubRelease{}, GitHubResponse{}, err
	}

	tagResp := GitHubResponse{response: gitHubTagListResponse}
	err = tagResp.Check()
	if err != nil {
		return GitHubRelease{}, tagResp, err
	}

	release := GitHubRelease{release: gitHubRelease}

	for _, tag := range tagList {
		if tag == nil {
			continue
		}
		if tag.GetName() != tagName {
			continue
		}
		release.repoTag = tag
		return release, GitHubResponse{response: gitHubResponse}, nil
	}

	return GitHubRelease{}, GitHubResponse{}, errors.New("Failed to create GitHub release: failed to locate tag")
}

func (client GitHubClient) CreateRelease(release Release) (Release, Response, error) {
	if client.client == nil {
		return GitHubRelease{}, GitHubResponse{}, errors.New("GitHub client is nil")
	}
	gitHubRelease, gitHubResponse, err := client.client.Repositories.CreateRelease(client.ctx, client.owner, client.repo, release.(GitHubRelease).release)
	return GitHubRelease{release: gitHubRelease}, GitHubResponse{response: gitHubResponse}, err
}

func (client GitHubClient) UpdateRelease(release Release) (Release, Response, error) {
	if client.client == nil {
		return GitHubRelease{}, GitHubResponse{}, errors.New("GitHub client is nil")
	}
	gitHubRelease, gitHubResponse, err := client.client.Repositories.EditRelease(client.ctx, client.owner, client.repo,
		release.GetID(), release.(GitHubRelease).release)
	return GitHubRelease{release: gitHubRelease}, GitHubResponse{response: gitHubResponse}, err
}

func (client GitHubClient) DeleteRelease(releaseId int) (Response, error) {
	if client.client == nil {
		return GitHubResponse{}, errors.New("GitHub client is nil")
	}
	gitHubResponse, err := client.client.Repositories.DeleteRelease(client.ctx, client.owner, client.repo, releaseId)
	return GitHubResponse{response: gitHubResponse}, err
}

func (client GitHubClient) DeleteTag(tagName string) (Response, error) {
	if client.client == nil {
		return GitHubResponse{}, errors.New("GitHub client is nil")
	}
	if client.httpClient == nil {
		return GitHubResponse{}, errors.New("GitHub http client is nil")
	}
	// GitHub guys haven't really created any actual API for tag deletion so need to do it the hard way
	deleteUrl := "https://api.github.com/repos/" + client.owner + "/" + client.repo + "/git/refs/tags/" + tagName
	request, err := http.NewRequest("DELETE", deleteUrl, nil)
	if err != nil {
		return GitHubResponse{}, err
	}

	gitHubResponse, err := client.httpClient.Do(request)
	return GitHubResponse{response: &github.Response{gitHubResponse, 0, 0, 0, 0, github.Rate{}}}, err
}

func (client GitHubClient) ListReleaseAssets(releaseId int) ([]ReleaseAsset, Response, error) {
	if client.client == nil {
		return nil, GitHubResponse{}, errors.New("GitHub client is nil")
	}
	gitHubReleaseAssets, gitHubResponse, err := client.client.Repositories.ListReleaseAssets(client.ctx, client.owner,
		client.repo, releaseId, nil)
	releaseAssets := make([]ReleaseAsset, 0, len(gitHubReleaseAssets))
	for _, gitHubReleaseAsset := range gitHubReleaseAssets {
		releaseAssets = append(releaseAssets, GitHubReleaseAsset{asset: gitHubReleaseAsset})
	}
	return releaseAssets, GitHubResponse{response: gitHubResponse}, err
}

func (client GitHubClient) DeleteReleaseAsset(assetId int) (Response, error) {
	if client.client == nil {
		return GitHubResponse{}, errors.New("GitHub client is nil")
	}
	gitHubResponse, err := client.client.Repositories.DeleteReleaseAsset(client.ctx, client.owner, client.repo, assetId)
	return GitHubResponse{response: gitHubResponse}, err
}

func (client GitHubClient) UploadReleaseAsset(releaseId int, assetName string,
	assetFile *os.File) (ReleaseAsset, Response, error) {
	if client.client == nil {
		return GitHubReleaseAsset{}, GitHubResponse{}, errors.New("GitHub client is nil")
	}
	var options github.UploadOptions
	options.Name = assetName
	gitHubReleaseAsset, gitHubResponse, err := client.client.Repositories.UploadReleaseAsset(client.ctx, client.owner,
		client.repo, releaseId, &options, assetFile)
	return GitHubReleaseAsset{asset: gitHubReleaseAsset}, GitHubResponse{response: gitHubResponse}, err
}

func (response GitHubResponse) Check() error {
	if response.response == nil {
		return errors.New("Response is nil")
	}

	if response.response.Response == nil {
		return errors.New("No HTTP response")
	}

	if response.GetStatusCode() < 200 || response.GetStatusCode() > 299 {
		return fmt.Errorf("Bad status code %d: %s\n", response.GetStatusCode(), response.GetStatus())
	}

	return nil
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
	return release.release.GetName()
}

func (release GitHubRelease) GetBody() string {
	if release.release == nil {
		return ""
	}
	return release.release.GetBody()
}

func (release GitHubRelease) SetBody(body string) {
	if release.release != nil {
		if release.release.Body == nil {
			release.release.Body = new(string)
		}
		*release.release.Body = body
	}
}

func (release GitHubRelease) GetTagName() string {
	if release.release == nil {
		return ""
	}
	return release.release.GetTagName()
}

func (release GitHubRelease) GetTargetCommitish() string {
	if release.repoTag == nil {
		return ""
	}
	commit := release.repoTag.GetCommit()
	if commit == nil {
		return ""
	}
	return commit.GetSHA()
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

func (release GitHubRelease) GetAssets() []ReleaseAsset {
	if release.release == nil {
		return nil
	}
	if release.release.Assets == nil {
		return nil
	}
	assets := make([]ReleaseAsset, 0, len(release.release.Assets))
	for _, gitHubReleaseAsset := range release.release.Assets {
		assets = append(assets, GitHubReleaseAsset{asset: &gitHubReleaseAsset})
	}
	return assets
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
