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
	DeleteRelease(release Release) (Response, error)
	DeleteTag(tagName string) (Response, error)
	DeleteReleaseAsset(asset ReleaseAsset) (Response, error)
	UploadReleaseAsset(release Release, assetName string, assetFile *os.File) (ReleaseAsset, Response, error)
}

type Release interface {
	GetName() string
	GetBody() string
	SetBody(body string)
	GetTagName() string
	GetTargetCommitish() string
	GetDraft() bool
	GetPrerelease() bool
	GetAssets() ([]ReleaseAsset, error)
}

type Response interface {
	Check() error
	GetStatusCode() int
	GetStatus() string
	GetBody() io.ReadCloser
	CloseBody()
}

type ReleaseAsset interface {
	GetTagName() string
	GetName() string
}
