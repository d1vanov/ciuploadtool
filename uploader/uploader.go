package uploader

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		fmt.Println("Creating new release")
		release, response, err = client.CreateRelease(releaseFactory(releaseBody, info))
	} else {
		existingReleaseAssets, response, err = client.ListReleaseAssets(release.GetID())
	}

	response.CloseBody()
	if err != nil {
		return client, err
	}

	err = response.Check()
	if err != nil {
		if !releaseExists {
			return client, fmt.Errorf("Bad response on attempt to create the new release: %v", err)
		}
		return client, fmt.Errorf("Bad response on attempt to list release assets: %v", err)
	}

	if releaseExists {
		release = updateBuildLogWithinReleaseBody(release, info)
		release, response, err = client.UpdateRelease(release)
		response.CloseBody()
		if err != nil {
			return client, err
		}
		fmt.Printf("Updated build log within the release body: %+v\n", release)
	} else {
		fmt.Printf("Created release: %+v\n", release)
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

func updateBuildLogWithinReleaseBody(release Release, info *buildEventInfo) Release {
	existingBody := release.GetBody()
	scanner := bufio.NewScanner(strings.NewReader(existingBody))
	newBody := ""
	foundCiLine := false
	for scanner.Scan() {
		line := scanner.Text()
		if info.isTravisCi && strings.HasPrefix(line, "Travis CI build log: https://travis-ci.org/"+info.owner+"/"+info.repo+"/builds/") {
			foundCiLine = true
			line = ciBuildLogString(info)
		} else if !info.isTravisCi && strings.HasPrefix(line, "AppVeyor CI build log: https://ci.appveyor.com/project/"+info.owner+"/"+info.repo+"/build") {
			foundCiLine = true
			line = ciBuildLogString(info)
		}
		newBody = newBody + line + "\n"
	}

	if !foundCiLine {
		newBody = newBody + ciBuildLogString(info) + "\n"
	}

	release.SetBody(newBody)
	return release
}

func ciBuildLogString(info *buildEventInfo) string {
	if len(info.buildId) == 0 {
		return ""
	}
	if info.isTravisCi {
		return "Travis CI build log: https://travis-ci.org/" + info.owner + "/" + info.repo + "/builds/" + info.buildId + "/"
	}
	return "AppVeyor CI build log: https://ci.appveyor.com/project/" + info.owner + "/" + info.repo + "/build/" + info.buildId + "/"
}
