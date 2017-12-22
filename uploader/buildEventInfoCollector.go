package uploader

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type buildEventInfo struct {
	token         string
	tag           string
	commit        string
	branch        string
	repo          string
	owner         string
	isPullRequest bool
	releaseTitle  string
	isPrerelease  bool
	isTravisCi    bool
	buildId       string
}

func collectBuildEventInfo(releaseSuffix string) (*buildEventInfo, error) {
	// Check whether the app is run during Travis CI or AppVeyor CI build
	appVeyorEnvVar := os.Getenv("APPVEYOR")
	travisCiEnvVar := os.Getenv("TRAVIS")
	isTravisCi := travisCiEnvVar == "true"
	isAppVeyor := appVeyorEnvVar == "True"
	if !isTravisCi && !isAppVeyor {
		fmt.Println("Neither Travis CI build nor AppVeyor build. Not doing anything")
		return nil, nil
	}

	var info buildEventInfo

	// Get GitHub API token from the environment variable
	if isAppVeyor {
		info.token = os.Getenv("auth_token")
	} else {
		info.token = os.Getenv("GITHUB_TOKEN")
	}

	if info.token == "" {
		if isAppVeyor {
			// This happens in AppVeyor CI on pull request builds, will silently
			// ignore that
			return nil, nil
		}

		return nil, errors.New("No GitHub access token, can't proceed")
	}

	// Get various build information from environment variables
	// specific to Travis CI and AppVeyor CI
	info.isTravisCi = isTravisCi

	repoSlug := ""

	if isAppVeyor {
		fmt.Println("Running on AppVeyor CI")
		info.branch = os.Getenv("APPVEYOR_REPO_BRANCH")
		info.tag = os.Getenv("APPVEYOR_REPO_TAG_NAME")
		info.commit = os.Getenv("APPVEYOR_REPO_COMMIT")
		repoSlug = os.Getenv("APPVEYOR_REPO_NAME")
		info.buildId = os.Getenv("APPVEYOR_BUILD_VERSION")
		info.isPullRequest = os.Getenv("APPVEYOR_PULL_REQUEST_NUMBER") != ""
	} else {
		fmt.Println("Running on Travis CI")
		info.branch = os.Getenv("TRAVIS_BRANCH")
		info.tag = os.Getenv("TRAVIS_TAG")
		info.commit = os.Getenv("TRAVIS_COMMIT")
		repoSlug = os.Getenv("TRAVIS_REPO_SLUG")
		info.buildId = os.Getenv("TRAVIS_BUILD_ID")
		info.isPullRequest = os.Getenv("TRAVIS_EVENT_TYPE") == "pull_request"
	}

	if info.isPullRequest {
		fmt.Println("Current build is the one triggered by a pull request, won't do anything")
		return nil, nil
	}

	if len(info.branch) == 0 {
		fmt.Println("No branch info was found, fallback to \"master\"")
		info.branch = "master"
	}

	fmt.Println("Commit: ", info.commit)

	repoSlugSplitted := strings.Split(repoSlug, "/")
	if len(repoSlugSplitted) != 2 {
		fmt.Printf("Error splitting repo slug into owner and repo: %s\n", repoSlug)
		os.Exit(-1)
	}

	info.owner = repoSlugSplitted[0]
	info.repo = repoSlugSplitted[1]

	if len(releaseSuffix) == 0 || releaseSuffix == info.tag {
		if len(info.tag) == 0 {
			fmt.Printf("No tag info was found, fallback to %q\n", info.branch)
			info.tag = info.branch
		}
		info.releaseTitle = "Release build (" + info.tag + ")"
	} else if len(releaseSuffix) != 0 {
		fmt.Printf("Suffix = %s\n", releaseSuffix)
		info.tag = "continuous-" + releaseSuffix
		info.releaseTitle = "Continuous build (" + info.tag + ")"
		info.isPrerelease = true
	} else {
		info.tag = "continuous" // Do not use "latest" as it is reserved by GitHub
		info.releaseTitle = "Continuous build"
		info.isPrerelease = true
	}

	return &info, nil
}
