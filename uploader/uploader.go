package uploader

import (
	"fmt"
	"os"
	"path/filepath"
)

type clientFactoryFunc func(gitHubToken string, owner string, repo string) Client

type releaseFactoryFunc func(releaseBody string, info *buildEventInfo) Release

func Upload(filenames []string, releaseSuffix string, releaseBody string) error {
	_, err := uploadImpl(clientFactoryFunc(newGitHubClient), newGitHubRelease, filenames, releaseSuffix, releaseBody)
	return err
}

func uploadImpl(clientFactory clientFactoryFunc, releaseFactory releaseFactoryFunc, filenames []string, releaseSuffix string, releaseBody string) (Client, error) {
	// Collect the information about the current build event
	info, err := collectBuildEventInfo(releaseSuffix)
	if err != nil {
		return nil, err
	}

	if info == nil {
		return nil, nil
	}

	client := clientFactory(info.token, info.owner, info.repo)

	// Check whether the release corresponding to the tag already exists
	releaseExists := false

	release, response, err := client.GetReleaseByTag(info.tag)
	response.CloseBody()
	if err != nil {
		if response != nil && response.GetStatusCode() == 404 {
			err = nil
		}
		if err != nil {
			err = fmt.Errorf("Failed to fetch release information: %v", err)
			return client, err
		}
	} else {
		err = response.Check()
		if err != nil {
			return client, err
		}
		releaseExists = true
	}

	if releaseExists {
		targetCommitish := release.GetTargetCommitish()
		if len(targetCommitish) != 0 && info.commit != targetCommitish {
			fmt.Printf("Found existing release but its commit SHA doesn't match the current one: %s vs %s\n", info.commit, targetCommitish)
			fmt.Printf("Deleting the existing release to recreate it with the current commit SHA %s\n", info.commit)

			response, err = client.DeleteRelease(release.GetID())
			response.CloseBody()
			if err != nil {
				return client, err
			}

			releaseExists = false

			if info.isPrerelease {
				fmt.Println("Since the existing release was pre-release one, need to also remove the tag corresponding to it")
				response, err = client.DeleteTag(info.tag)
				response.CloseBody()
				if err != nil {
					return client, err
				}
			}
		}
	}

	var existingReleaseAssets []ReleaseAsset

	if !releaseExists {
		release, response, err = client.CreateRelease(releaseFactory(releaseBody, info))
	} else {
		existingReleaseAssets, response, err = client.ListReleaseAssets(release.GetID())
	}

	response.CloseBody()
	if err != nil {
		return client, err
	}

	for _, filename := range commandLineFiles(filenames) {
		file, err := os.Open(filename)
		if err != nil {
			return client, err
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			return client, err
		}

		mode := stat.Mode()
		if !mode.IsRegular() {
			fmt.Printf("Skipping dir %s\n", filename)
			continue
		}

		if existingReleaseAssets != nil {
			for i, existingReleaseAsset := range existingReleaseAssets {
				if existingReleaseAsset.GetID() == 0 {
					continue
				}
				if len(existingReleaseAsset.GetName()) == 0 {
					continue
				}
				if existingReleaseAsset.GetName() == filepath.Base(filename) {
					fmt.Printf("Found duplicate release asset %s, deleting it\n", existingReleaseAsset.GetName())
					response, err = client.DeleteReleaseAsset(existingReleaseAsset.GetID())
					response.CloseBody()
					if err != nil {
						return client, err
					}

					err = response.Check()
					if err != nil {
						return client, fmt.Errorf("Bad response on attempt to delete the stale release asset: %v", err)
					}

					existingReleaseAssets = append(existingReleaseAssets[:i], existingReleaseAssets[i+1:]...)
				}
			}
		}

		fmt.Printf("Trying to upload file: %s\n", filename)

		asset, response, err := client.UploadReleaseAsset(release.GetID(), filepath.Base(filename), file)
		response.CloseBody()
		if err != nil {
			return client, err
		}

		err = response.Check()
		if err != nil {
			return client, fmt.Errorf("Bad response on attempt to upload release asset: %v", err)
		}

		existingReleaseAssets = append(existingReleaseAssets, asset)
	}

	return client, nil
}

func appendCiLogToReleaseBody(release Release, info *buildEventInfo) {
	existingBody := release.GetBody()
	if info.isTravisCi && len(info.buildId) != 0 {
		existingBody = existingBody + "\nTravis CI build log: https://travis-ci.org/" + info.owner + "/" + info.repo + "/builds/" + info.buildId + "/"
	}
}
