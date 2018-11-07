package uploader

import (
	"context"
	"io"
	"os"
)

type Client interface {
	GetContext() context.Context
	GetOwner() string
	GetRepo() string
	GetReleaseByTag(tagName string) (Release, Response, error)
	CreateRelease(release Release) (Release, Response, error)
	UpdateRelease(release Release) (Release, Response, error)
	DeleteRelease(releaseId int64) (Response, error)
	DeleteTag(tagName string) (Response, error)
	ListReleaseAssets(releaseId int64) ([]ReleaseAsset, Response, error)
	DeleteReleaseAsset(assetId int64) (Response, error)
	UploadReleaseAsset(releaseId int64, assetName string, assetFile *os.File) (ReleaseAsset, Response, error)
}

type Release interface {
	GetID() int64
	GetName() string
	GetBody() string
	SetBody(body string)
	GetTagName() string
	GetTargetCommitish() string
	GetDraft() bool
	GetPrerelease() bool
	GetAssets() []ReleaseAsset
}

type Response interface {
	Check() error
	GetStatusCode() int
	GetStatus() string
	GetBody() io.ReadCloser
	CloseBody()
}

type ReleaseAsset interface {
	GetID() int64
	GetName() string
}
