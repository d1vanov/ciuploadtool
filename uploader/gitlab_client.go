package uploader

import (
	"context"
	"errors"
	"fmt"
	"github.com/xanzy/go-gitlab"
	"io"
	"os"
)

type GitLabClient struct {
	client *gitlab.Client
	ctx    context.Context
	owner  string
	repo   string
}

type GitLabResponse struct {
	response *gitlab.Response
}

type GitLabRelease struct {
	release      *gitlab.Release
	commit       string
	isPrerelease bool
}

type GitLabReleaseAsset struct {
	tagName string
}

func newGitLabClient(gitLabToken string, owner string, repo string) Client {
	client := gitlab.NewClient(nil, gitLabToken)
	return GitLabClient{client: client, owner: owner, repo: repo}
}

func newGitLabRelease(releaseBody string, info *buildEventInfo) Release {
	release := GitLabRelease{release: new(gitlab.Release)}
	release.release.TagName = info.tag
	release.commit = info.commit
	release.release.Description = releaseBody
	release.isPrerelease = info.isPrerelease
	return updateBuildLogWithinReleaseBody(release, info)
}

func (client GitLabClient) GetContext() context.Context {
	return client.ctx
}

func (client GitLabClient) GetOwner() string {
	return client.owner
}

func (client GitLabClient) GetRepo() string {
	return client.repo
}

func (client GitLabClient) GetReleaseByTag(tagName string) (Release, Response, error) {
	if client.client == nil {
		return GitLabRelease{}, GitLabResponse{}, errors.New("GitLab client is nil")
	}
	if client.client.Tags == nil {
		return GitLabRelease{}, GitLabResponse{}, errors.New("GitLab client.Tags is nil")
	}
	pid := client.owner + "%2F" + client.repo
	gitLabTag, gitLabResponse, err := client.client.Tags.GetTag(pid, tagName)
	return GitLabRelease{release: gitLabTag.Release}, GitLabResponse{response: gitLabResponse}, err
}

func (client GitLabClient) CreateRelease(release Release) (Release, Response, error) {
	if client.client == nil {
		return GitLabRelease{}, GitLabResponse{}, errors.New("GitLab client is nil")
	}
	if client.client.Tags == nil {
		return GitLabRelease{}, GitLabResponse{}, errors.New("GitLab client.Tags is nil")
	}
	pid := client.owner + "%2F" + client.repo
	body := release.GetBody()
	gitLabRelease, gitLabResponse, err := client.client.Tags.CreateRelease(pid, release.GetTagName(), &gitlab.CreateReleaseOptions{Description: &body})
	return GitLabRelease{release: gitLabRelease}, GitLabResponse{response: gitLabResponse}, err
}

func (client GitLabClient) UpdateRelease(release Release) (Release, Response, error) {
	if client.client == nil {
		return GitLabRelease{}, GitLabResponse{}, errors.New("GitLab client is nil")
	}
	if client.client.Tags == nil {
		return GitLabRelease{}, GitLabResponse{}, errors.New("GitLab client.Tags is nil")
	}
	pid := client.owner + "%2F" + client.repo
	body := release.GetBody()
	gitLabRelease, gitLabResponse, err := client.client.Tags.UpdateRelease(pid, release.GetTagName(), &gitlab.UpdateReleaseOptions{Description: &body})
	return GitLabRelease{release: gitLabRelease}, GitLabResponse{response: gitLabResponse}, err
}

func (client GitLabClient) DeleteRelease(release Release) (Response, error) {
	if client.client == nil {
		return GitLabResponse{}, errors.New("GitLab client is nil")
	}
	if client.client.Tags == nil {
		return GitLabResponse{}, errors.New("GitLab client.Tags is nil")
	}
	pid := client.owner + "%2F" + client.repo
	tagName := release.GetTagName()
	// Have to do it in two steps: remove the entire tag and then create it once again
	// But first get the current tag
	tag, getTagResponse, err := client.client.Tags.GetTag(pid, tagName)
	if err != nil {
		return GitLabResponse{}, fmt.Errorf("Failed to get GitLab tag by name: %v", err)
	}
	err = GitLabResponse{response: getTagResponse}.Check()
	if err != nil {
		return GitLabResponse{response: getTagResponse}, nil
	}
	// Now delete the tag
	deleteTagResponse, err := client.DeleteTag(tagName)
	if err != nil {
		return GitLabResponse{}, err
	}
	err = GitLabResponse{response: deleteTagResponse.(GitLabResponse).response}.Check()
	if err != nil {
		return GitLabResponse{response: deleteTagResponse.(GitLabResponse).response}, nil
	}
	// Now create the tag once again
	body := release.GetBody()
	_, createTagResponse, err := client.client.Tags.CreateTag(pid, &gitlab.CreateTagOptions{TagName: &tagName, Message: &tag.Message, ReleaseDescription: &body})
	return GitLabResponse{response: createTagResponse}, err
}

func (client GitLabClient) DeleteTag(tagName string) (Response, error) {
	if client.client == nil {
		return GitLabResponse{}, errors.New("GitLab client is nil")
	}
	if client.client.Tags == nil {
		return GitLabResponse{}, errors.New("GitLab client.Tags is nil")
	}
	pid := client.owner + "%2F" + client.repo
	deleteTagResponse, err := client.client.Tags.DeleteTag(pid, tagName)
	return GitLabResponse{response: deleteTagResponse}, err
}

func (client GitLabClient) DeleteReleaseAsset(asset ReleaseAsset) (Response, error) {
	// TODO: implement
	return GitLabResponse{}, nil
}

func (client GitLabClient) UploadReleaseAsset(release Release, assetName string, assetFile *os.File) (ReleaseAsset, Response, error) {
	// TODO: implement
	return GitLabReleaseAsset{}, GitLabResponse{}, nil
}

func (response GitLabResponse) Check() error {
	if response.response == nil {
		return errors.New("Response is nil")
	}

	if response.GetStatusCode() < 200 || response.GetStatusCode() > 299 {
		return fmt.Errorf("Bad status code %d: %s\n", response.GetStatusCode(), response.GetStatus())
	}

	return nil
}

func (response GitLabResponse) GetStatusCode() int {
	if response.response == nil {
		return -1
	}
	return response.response.StatusCode
}

func (response GitLabResponse) GetStatus() string {
	if response.response == nil {
		return ""
	}
	return response.response.Status
}

func (response GitLabResponse) GetBody() io.ReadCloser {
	if response.response == nil {
		return nil
	}
	return response.response.Body
}

func (response GitLabResponse) CloseBody() {
	if response.response == nil {
		return
	}
	response.response.Body.Close()
}

func (release GitLabRelease) GetName() string {
	if release.release == nil {
		return ""
	}
	return release.release.TagName
}

func (release GitLabRelease) GetBody() string {
	if release.release == nil {
		return ""
	}
	return release.release.Description
}

func (release GitLabRelease) SetBody(body string) {
	if release.release != nil {
		release.release.Description = body
	}
}

func (release GitLabRelease) GetTagName() string {
	if release.release == nil {
		return ""
	}
	return release.release.TagName
}

func (release GitLabRelease) GetTargetCommitish() string {
	if release.release == nil {
		return ""
	}
	return release.commit
}

func (release GitLabRelease) GetDraft() bool {
	return false
}

func (release GitLabRelease) GetPrerelease() bool {
	return release.isPrerelease
}

func (release GitLabRelease) GetAssets() []ReleaseAsset {
	if release.release == nil {
		return nil
	}
	// TODO: parse the release's description and extract URLs of release assets from it
	return nil
}

func (releaseAsset GitLabReleaseAsset) GetTagName() string {
	return releaseAsset.tagName
}

func (releaseAsset GitLabReleaseAsset) GetName() string {
	// TODO: implement somehow
	return ""
}
