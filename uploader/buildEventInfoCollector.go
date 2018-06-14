package uploader

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	gitlabCi   = iota
	travisCi   = iota
	appveyorCi = iota
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
	whichCi       int
	buildId       string
}

func collectBuildEventInfo(args *uploadArgs) (*buildEventInfo, error) {
	// Check whether the app is run during Travis CI or AppVeyor CI or GitLab CI build
	appVeyorEnvVar := os.Getenv("APPVEYOR")
	travisCiEnvVar := os.Getenv("TRAVIS")
	gitLabCiEnvVar := os.Getenv("GITLAB_CI")
	isTravisCi := travisCiEnvVar == "true"
	isAppVeyor := appVeyorEnvVar == "True"
	isGitLabCi := gitLabCiEnvVar == "true"
	if !isTravisCi && !isAppVeyor && !isGitLabCi {
		fmt.Println("Neither Travis CI build nor AppVeyor build nor GitLab CI build. Not doing anything")
		return nil, nil
	}

	var info buildEventInfo

	// Get GitHub/GitLab API token from the environment variable
	if isAppVeyor {
		info.token = os.Getenv("auth_token")
	} else if isTravisCi {
		info.token = os.Getenv("GITHUB_TOKEN")
	} else {
		info.token = os.Getenv("GITLAB_TOKEN")
	}

	if info.token == "" {
		if isAppVeyor {
			fmt.Println("No dev token for AppVeyor CI job, won't do anything")
			// This happens in AppVeyor CI on pull request builds, will silently
			// ignore that
			return nil, nil
		}

		return nil, errors.New("No GitHub/GitLab access token, can't proceed")
	}

	// Get various build information from environment variables
	// specific to Travis CI and AppVeyor CI and GitLab CI
	if isAppVeyor {
		info.whichCi = appveyorCi
	} else if isTravisCi {
		info.whichCi = travisCi
	} else {
		info.whichCi = gitlabCi
	}

	repoSlug := ""

	if isAppVeyor {
		fmt.Println("Running on AppVeyor CI")
		info.branch = os.Getenv("APPVEYOR_REPO_BRANCH")
		info.tag = os.Getenv("APPVEYOR_REPO_TAG_NAME")
		info.commit = os.Getenv("APPVEYOR_REPO_COMMIT")
		repoSlug = os.Getenv("APPVEYOR_REPO_NAME")
		info.buildId = os.Getenv("APPVEYOR_BUILD_VERSION")
		info.isPullRequest = os.Getenv("APPVEYOR_PULL_REQUEST_NUMBER") != ""
	} else if isTravisCi {
		fmt.Println("Running on Travis CI")
		info.branch = os.Getenv("TRAVIS_BRANCH")
		info.tag = os.Getenv("TRAVIS_TAG")
		info.commit = os.Getenv("TRAVIS_COMMIT")
		repoSlug = os.Getenv("TRAVIS_REPO_SLUG")
		info.buildId = os.Getenv("TRAVIS_BUILD_ID")
		info.isPullRequest = os.Getenv("TRAVIS_EVENT_TYPE") == "pull_request"
	} else {
		fmt.Println("Running on GitLab CI")
		info.branch = os.Getenv("CI_COMMIT_REF_NAME")
		info.tag = os.Getenv("CI_COMMIT_TAG")
		info.commit = os.Getenv("CI_COMMIT_SHA")
		repoSlug = os.Getenv("CI_PROJECT_NAMESPACE") + "/" + os.Getenv("CI_PROJECT_NAME")
		info.buildId = os.Getenv("CI_JOB_ID")
		info.isPullRequest = false
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

	if len(info.tag) != 0 && !strings.HasPrefix(info.tag, "continuous") && (len(args.releaseSuffix) == 0 || args.releaseSuffix == info.tag) {
		info.releaseTitle = "Release build (" + info.tag + ")"
	} else if len(args.releaseSuffix) != 0 {
		fmt.Printf("Suffix = %s\n", args.releaseSuffix)
		info.tag = "continuous-" + args.releaseSuffix
		info.releaseTitle = "Continuous build (" + info.tag + ")"
		info.isPrerelease = true
	} else {
		info.tag = "continuous" // Do not use "latest" as it is reserved by GitHub
		info.releaseTitle = "Continuous build"
		info.isPrerelease = true
	}

	return &info, nil
}
