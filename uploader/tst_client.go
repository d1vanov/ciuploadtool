package uploader

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
)

type TstClient struct {
	token    string
	releases *[]TstRelease
	tagNames []string
}

type TstResponse struct {
	statusCode int
	status     string
	body       bytes.Buffer
}

type TstRelease struct {
	id              int
	name            string
	tagName         string
	targetCommitish string
	isDraft         bool
	isPrerelease    bool
	assets          []TstReleaseAsset
}

type TstReleaseAsset struct {
	id   int
	name string
}

func (client TstClient) GetReleaseByTag(ctx context.Context, owner string, repo string, tagName string) (TstRelease, TstResponse, error) {
	if client.releases == nil {
		return TstRelease{}, TstResponse{statusCode: 404, status: "Not found"}, errors.New("No releases within the test client")
	}
	if len(client.token) == 0 {
		return TstRelease{}, TstResponse{statusCode: 401, status: "Bad credentials"}, errors.New("No GitHub token")
	}
	for _, release := range *client.releases {
		if release.tagName == tagName {
			return release, TstResponse{statusCode: 200}, nil
		}
	}
	return TstRelease{}, TstResponse{statusCode: 404, status: "Not found"}, errors.New("Release matching tag name was not found")
}

func (client TstClient) CreateRelease(ctx context.Context, owner string, repo string, release Release) (TstRelease, TstResponse, error) {
	if len(client.token) == 0 {
		return TstRelease{}, TstResponse{statusCode: 401, status: "Bad credentials"}, errors.New("No GitHub token")
	}
	tagName := release.GetTagName()
	if len(tagName) == 0 {
		return TstRelease{}, TstResponse{statusCode: 400, status: "Missing tag name"}, errors.New("The release to be created has no tag name")
	}
	name := release.GetName()
	if len(name) == 0 {
		return TstRelease{}, TstResponse{statusCode: 400, status: "Missing release name"}, errors.New("The release to be created has no name")
	}
	if client.releases == nil {
		client.releases = &[]TstRelease{}
	}
	*client.releases = append(*client.releases, release.(TstRelease))
	client.DeleteTag(ctx, owner, repo, tagName)
	client.tagNames = append(client.tagNames, tagName)
	return release.(TstRelease), TstResponse{statusCode: 200, status: "Created"}, nil
}

func (client TstClient) DeleteRelease(ctx context.Context, owner string, repo string, releaseId int) (TstResponse, error) {
	if len(client.token) == 0 {
		return TstResponse{statusCode: 401, status: "Bad credentials"}, errors.New("No GitHub token")
	}
	if client.releases == nil {
		return TstResponse{statusCode: 204, status: "No content"}, errors.New("No releases within client")
	}
	for _, release := range *client.releases {
		if release.GetID() == releaseId {
			return TstResponse{statusCode: 204, status: "No content"}, nil
		}
	}
	return TstResponse{statusCode: 204, status: "No content"}, errors.New("No such release found")
}

func (client TstClient) DeleteTag(ctx context.Context, owner string, repo string, tagName string) (TstResponse, error) {
	if len(client.token) == 0 {
		return TstResponse{statusCode: 401, status: "Bad credentials"}, errors.New("No GitHub token")
	}
	for i, clientTagName := range client.tagNames {
		if clientTagName == tagName {
			client.tagNames = append(client.tagNames[:i], client.tagNames[i+1:]...)
			return TstResponse{statusCode: 200, status: "Deleted"}, nil
		}
	}
	return TstResponse{statusCode: 404, status: "Not found"}, errors.New("Found no tag to delete")
}

func (response TstResponse) GetStatusCode() int {
	return response.statusCode
}

func (response TstResponse) GetStatus() string {
	return response.status
}

func (response TstResponse) GetBody() io.ReadCloser {
	return ioutil.NopCloser(&response.body)
}

func (response TstResponse) CloseBody() {
}

func (release TstRelease) GetID() int {
	return release.id
}

func (release TstRelease) GetName() string {
	return release.name
}

func (release TstRelease) GetTagName() string {
	return release.tagName
}

func (release TstRelease) GetTargetCommitish() string {
	return release.targetCommitish
}

func (release TstRelease) GetDraft() bool {
	return release.isDraft
}

func (release TstRelease) GetPrerelease() bool {
	return release.isPrerelease
}

func (releaseAsset TstReleaseAsset) GetID() int {
	return releaseAsset.id
}

func (releaseAsset TstReleaseAsset) GetName() string {
	return releaseAsset.name
}
