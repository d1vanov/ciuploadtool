package uploader

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type clientFactoryFunc func(authToken string, owner string, repo string) Client

type releaseFactoryFunc func(releaseBody string, info *buildEventInfo) Release

type overrideUploadOptions struct {
	service string
	repo    string
	owner   string
	token   string
}

type uploadArgs struct {
	clientFactory   clientFactoryFunc
	releaseFactory  releaseFactoryFunc
	filenames       []string
	releaseSuffix   string
	releaseBody     string
	overrideOptions overrideUploadOptions
}

func Upload(filenames []string, uploadToService, uploadToRepo, uploadToRepoOwner, uploadAuthToken, releaseSuffix, releaseBody string) error {
	var args uploadArgs
	args.filenames = filenames
	args.releaseSuffix = releaseSuffix
	args.releaseBody = releaseBody
	args.overrideOptions.service = uploadToService
	args.overrideOptions.repo = uploadToRepo
	args.overrideOptions.owner = uploadToRepoOwner
	args.overrideOptions.token = uploadAuthToken
	_, err := uploadImpl(&args)
	return err
}

func uploadImpl(args *uploadArgs) (Client, error) {
	// Collect the information about the current build event
	info, err := collectBuildEventInfo(args)
	if err != nil {
		return nil, err
	}

	if info == nil {
		fmt.Println("No build event info, won't do anything")
		return nil, nil
	}

	var token string
	if len(args.overrideOptions.token) != 0 {
		token = args.overrideOptions.token
	} else {
		token = info.token
	}

	var owner string
	if len(args.overrideOptions.owner) != 0 {
		owner = args.overrideOptions.owner
	} else {
		owner = info.owner
	}

	var repo string
	if len(args.overrideOptions.repo) != 0 {
		repo = args.overrideOptions.repo
	} else {
		repo = info.repo
	}

	if args.clientFactory == nil && args.releaseFactory == nil {
		if args.overrideOptions.service == "gitlab" || info.whichCi == gitlabCi {
			args.clientFactory = clientFactoryFunc(newGitLabClient)
			args.releaseFactory = releaseFactoryFunc(newGitLabRelease)
		} else {
			args.clientFactory = clientFactoryFunc(newGitHubClient)
			args.releaseFactory = releaseFactoryFunc(newGitHubRelease)
		}
	}

	client := args.clientFactory(token, owner, repo)

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

			response, err = client.DeleteRelease(release)
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
		release, response, err = client.CreateRelease(args.releaseFactory(args.releaseBody, info))
		response.CloseBody()
		if err != nil {
			return client, err
		}
		err = response.Check()
		if err != nil {
			return client, fmt.Errorf("Bad response on attempt to create the new release: %v", err)
		}
	} else {
		existingReleaseAssets, err = release.GetAssets()
		if err != nil {
			return client, err
		}
	}

	if releaseExists {
		release = updateBuildLogWithinReleaseBody(release, info)
		release, response, err = client.UpdateRelease(release)
		response.CloseBody()
		if err != nil {
			return client, err
		}
	} else {
		fmt.Println("Created new release")
	}

	for _, filename := range commandLineFiles(args.filenames) {
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
				if len(existingReleaseAsset.GetName()) == 0 {
					continue
				}
				if existingReleaseAsset.GetName() == filepath.Base(filename) {
					fmt.Printf("Found duplicate release asset %s, deleting it\n", existingReleaseAsset.GetName())
					response, err = client.DeleteReleaseAsset(existingReleaseAsset)
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

		asset, response, err := client.UploadReleaseAsset(release, filepath.Base(filename), file)
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
		if (info.whichCi == travisCi) && strings.HasPrefix(line, "Travis CI build log: https://travis-ci.org/"+info.owner+"/"+info.repo+"/builds/") {
			foundCiLine = true
			line = ciBuildLogString(info)
		} else if (info.whichCi == appveyorCi) && strings.HasPrefix(line, "AppVeyor CI build log: https://ci.appveyor.com/project/"+info.owner+"/"+info.repo+"/build") {
			foundCiLine = true
			line = ciBuildLogString(info)
		} else if (info.whichCi == gitlabCi) && strings.HasPrefix(line, "GitLab CI build log: https://gitlab.com/"+info.owner+"/"+info.repo+"/builds/") {
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
	if info.whichCi == travisCi {
		return "Travis CI build log: https://travis-ci.org/" + info.owner + "/" + info.repo + "/builds/" + info.buildId + "/"
	} else if info.whichCi == appveyorCi {
		return "AppVeyor CI build log: https://ci.appveyor.com/project/" + info.owner + "/" + info.repo + "/build/" + info.buildId
	} else {
		return "GitLab CI build log: https://gitlab.com/" + info.owner + "/" + info.repo + "/builds/" + info.buildId
	}
}
