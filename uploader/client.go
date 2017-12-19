package uploader

import (
	"context"
	"io"
	"os"
)

type Client interface {
	GetReleaseByTag(ctx context.Context, owner string, repo string, tagName string) (Release, Response, error)
	CreateRelease(ctx context.Context, owner string, repo string, release Release) (Release, Response, error)
	DeleteRelease(ctx context.Context, owner string, repo string, releaseId int) (Response, error)
	DeleteTag(ctx context.Context, owner string, repo string, tagName string) (Response, error)
	ListReleaseAssets(ctx context.Context, owner string, repo string, releaseId int) ([]ReleaseAsset, Response, error)
	DeleteReleaseAsset(ctx context.Context, owner string, repo string, assetId int) (Response, error)
	UploadReleaseAsset(ctx context.Context, owner string, repo string, releaseId int, assetName string,
		assetFile *os.File) (ReleaseAsset, Response, error)
}

type Release interface {
	GetID() int
	GetName() string
	GetTagName() string
	GetTargetCommitish() string
	GetDraft() bool
	GetPrerelease() bool
}

type Response interface {
	GetStatusCode() int
	GetStatus() string
	GetBody() io.ReadCloser
	CloseBody()
}

type ReleaseAsset interface {
	GetID() int
	GetName() string
}
