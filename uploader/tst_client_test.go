package uploader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

var lastFreeReleaseAssetId int64
var lastFreeReleaseId int64

func init() {
	lastFreeReleaseAssetId = 1
	lastFreeReleaseId = 1
}

type TstClient struct {
	token    string
	owner    string
	repo     string
	releases []TstRelease
	tagNames []string
}

type TstResponse struct {
	statusCode int
	status     string
	body       bytes.Buffer
}

type TstRelease struct {
	id              int64
	name            string
	body            string
	tagName         string
	targetCommitish string
	isDraft         bool
	isPrerelease    bool
	assets          []TstReleaseAsset
}

type TstReleaseAsset struct {
	id      int64
	name    string
	content string
}

func newTstClient(gitHubToken string, owner string, repo string) Client {
	return &TstClient{token: gitHubToken, owner: owner, repo: repo}
}

func newTstRelease(releaseBody string, info *buildEventInfo) Release {
	release := TstRelease{
		id:              lastFreeReleaseId,
		name:            info.releaseTitle,
		body:            releaseBody,
		tagName:         info.tag,
		targetCommitish: info.commit,
		isDraft:         false,
		isPrerelease:    info.isPrerelease,
	}
	lastFreeReleaseId++
	return updateBuildLogWithinReleaseBody(&release, info)
}

func (client TstClient) GetContext() context.Context {
	return context.Background()
}

func (client *TstClient) GetOwner() string {
	return client.owner
}

func (client *TstClient) GetRepo() string {
	return client.repo
}

func (client *TstClient) GetReleaseByTag(tagName string) (Release, Response, error) {
	if len(client.releases) == 0 {
		return nil, TstResponse{statusCode: 404, status: "Not found"}, errors.New("No releases within the test client")
	}
	if len(client.token) == 0 {
		return nil, TstResponse{statusCode: 401, status: "Bad credentials"}, errors.New("No GitHub token")
	}
	for _, release := range client.releases {
		if release.tagName == tagName {
			return &release, TstResponse{statusCode: 200}, nil
		}
	}
	return nil, TstResponse{statusCode: 404, status: "Not found"}, errors.New("Release matching tag name was not found")
}

func (client *TstClient) CreateRelease(release Release) (Release, Response, error) {
	if len(client.token) == 0 {
		return nil, TstResponse{statusCode: 401, status: "Bad credentials"}, errors.New("No GitHub token")
	}
	tagName := release.GetTagName()
	if len(tagName) == 0 {
		return nil, TstResponse{statusCode: 400, status: "Missing tag name"}, errors.New("The release to be created has no tag name")
	}
	name := release.GetName()
	if len(name) == 0 {
		return nil, TstResponse{statusCode: 400, status: "Missing release name"}, errors.New("The release to be created has no name")
	}

	tstRelease := release.(*TstRelease)
	tstRelease.id = lastFreeReleaseId
	lastFreeReleaseId++
	client.releases = append(client.releases, *tstRelease)

	client.DeleteTag(tagName)
	client.tagNames = append(client.tagNames, tagName)

	return tstRelease, TstResponse{statusCode: 200, status: "Created"}, nil
}

func (client *TstClient) UpdateRelease(release Release) (Release, Response, error) {
	if len(client.token) == 0 {
		return nil, TstResponse{statusCode: 401, status: "Bad credentials"}, errors.New("No GitHub token")
	}
	for i := range client.releases {
		if client.releases[i].GetID() != release.GetID() {
			continue
		}
		client.releases[i] = *(release.(*TstRelease))
		return release, TstResponse{statusCode: 200, status: "Updated"}, nil
	}
	return nil, TstResponse{statusCode: 404, status: "Not found"}, errors.New("Release matching by ID was not found")
}

func (client *TstClient) DeleteRelease(releaseId int64) (Response, error) {
	if len(client.token) == 0 {
		return TstResponse{statusCode: 401, status: "Bad credentials"}, errors.New("No GitHub token")
	}
	if len(client.releases) == 0 {
		return TstResponse{statusCode: 204, status: "No content"}, errors.New("No releases within client")
	}
	for i, release := range client.releases {
		if release.GetID() == releaseId {
			client.releases = append(client.releases[:i], client.releases[i+1:]...)
			return TstResponse{statusCode: 200, status: "Deleted"}, nil
		}
	}
	return TstResponse{statusCode: 204, status: "No content"}, errors.New("No such release found")
}

func (client *TstClient) DeleteTag(tagName string) (Response, error) {
	if len(client.token) == 0 {
		return TstResponse{statusCode: 401, status: "Bad credentials"}, errors.New("No GitHub token")
	}
	if len(client.tagNames) == 0 {
		return TstResponse{statusCode: 404, status: "Not found"}, errors.New("No tags within client")
	}
	for i, clientTagName := range client.tagNames {
		if clientTagName == tagName {
			client.tagNames = append(client.tagNames[:i], client.tagNames[i+1:]...)
			return TstResponse{statusCode: 200, status: "Deleted"}, nil
		}
	}
	return TstResponse{statusCode: 404, status: "Not found"}, errors.New("Found no tag to delete")
}

func (client *TstClient) ListReleaseAssets(releaseId int64) ([]ReleaseAsset, Response, error) {
	if len(client.token) == 0 {
		return nil, TstResponse{statusCode: 401, status: "Bad credentials"}, errors.New("No GitHub token")
	}
	if len(client.releases) == 0 {
		return nil, TstResponse{statusCode: 404, status: "Not found"}, errors.New("No releases within client")
	}
	for _, release := range client.releases {
		if release.GetID() == releaseId {
			return release.GetAssets(), TstResponse{statusCode: 200, status: "Found"}, nil
		}
	}
	return nil, TstResponse{statusCode: 404, status: "Not found"}, errors.New("Release with given id was not found")
}

func (client *TstClient) DeleteReleaseAsset(assetId int64) (Response, error) {
	if len(client.token) == 0 {
		return TstResponse{statusCode: 401, status: "Bad credentials"}, errors.New("No GitHub token")
	}
	if len(client.releases) == 0 {
		return TstResponse{statusCode: 404, status: "Not found"}, errors.New("No releases within client")
	}
	for i := range client.releases {
		for j, asset := range client.releases[i].GetAssets() {
			if asset.GetID() == assetId {
				client.releases[i].assets = append(client.releases[i].assets[:j], client.releases[i].assets[j+1:]...)
				return TstResponse{statusCode: 200, status: "Deleted"}, nil
			}
		}
	}
	return TstResponse{statusCode: 404, status: "Not found"}, errors.New("Release containing the asset with given id was not found")
}

func (client *TstClient) UploadReleaseAsset(releaseId int64, assetName string, assetFile *os.File) (ReleaseAsset, Response, error) {
	if len(client.token) == 0 {
		return TstReleaseAsset{}, TstResponse{statusCode: 401, status: "Bad credentials"}, errors.New("No GitHub token")
	}
	if len(client.releases) == 0 {
		return TstReleaseAsset{}, TstResponse{statusCode: 404, status: "Not found"}, errors.New("No releases within client")
	}
	for i, release := range client.releases {
		if release.GetID() == releaseId {
			for _, asset := range release.GetAssets() {
				if asset.GetName() == assetName {
					return TstReleaseAsset{}, TstResponse{statusCode: 400, status: "Release asset already exists"},
						errors.New("Release asset with the given name already exists")
				}
			}
			assetFileContent, err := ioutil.ReadAll(assetFile)
			if err != nil {
				return TstReleaseAsset{}, TstResponse{statusCode: 400, status: "Failed to read the asset file's contents"},
					fmt.Errorf("Failed to read the asset file's contents: %v", err)
			}
			asset := TstReleaseAsset{id: lastFreeReleaseAssetId, name: assetName, content: string(assetFileContent)}
			lastFreeReleaseAssetId++
			release.assets = append(release.assets, asset)
			client.releases[i] = release
			return asset, TstResponse{statusCode: 200, status: "Uploaded"}, nil
		}
	}
	return TstReleaseAsset{}, TstResponse{statusCode: 404, status: "Not found"}, errors.New("Release with given id was not found")
}

func (response TstResponse) Check() error {
	if response.GetStatusCode() < 200 || response.GetStatusCode() > 299 {
		return fmt.Errorf("Bad status code %d: %s\n", response.GetStatusCode(), response.GetStatus())
	}
	return nil
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

func (release *TstRelease) GetID() int64 {
	return release.id
}

func (release *TstRelease) GetName() string {
	return release.name
}

func (release *TstRelease) GetBody() string {
	return release.body
}

func (release *TstRelease) SetBody(body string) {
	release.body = body
}

func (release *TstRelease) GetTagName() string {
	return release.tagName
}

func (release *TstRelease) GetTargetCommitish() string {
	return release.targetCommitish
}

func (release *TstRelease) GetDraft() bool {
	return release.isDraft
}

func (release *TstRelease) GetPrerelease() bool {
	return release.isPrerelease
}

func (release *TstRelease) GetAssets() []ReleaseAsset {
	assets := make([]ReleaseAsset, 0, len(release.assets))
	for _, asset := range release.assets {
		assets = append(assets, asset)
	}
	return assets
}

func (releaseAsset TstReleaseAsset) GetID() int64 {
	return releaseAsset.id
}

func (releaseAsset TstReleaseAsset) GetName() string {
	return releaseAsset.name
}

func (releaseAsset TstReleaseAsset) GetContent() string {
	return releaseAsset.content
}
