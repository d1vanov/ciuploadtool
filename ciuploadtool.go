package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	var pSuffix = flag.String("suffix", "", "Optional suffix for the names of created continuous releases")
	var pReleaseBody = flag.String("relbody", "", "Optional content for the body of created releases")
	flag.Parse()

	if len(flag.NArg()) < 2 {
		fmt.Printf("Usage: %s [-suffix=<suffix for continuous release names>] [-relbody=<release body message>] <files to upload>\n", os.Args[0])
		os.Exit(-1)
	}

	info, err := getBuildEventInfo(pSuffix)
	if err != nil {
		fmt.Print(err)
		os.Exit(-1)
	}

	if info == nil {
		os.Exit(0)
	}

	// Setup GitHub client
	ctx = context.Background()
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tokenizedClient := oauth2.NewClient(ctx, tokenSource)

	client := github.NewClient(tokenizedClient)

	// Fetch release info
	release, response, err := client.Repositories.GetReleaseByTag(ctx, owner, repo, releaseName)
	if err != nil {
		fmt.Printf("Failed to request release information: %v", err)
		os.Exit(-1)
	}

	if response != nil {
		defer response.Body.Close()
	}

	err = checkResponse(response)
	if err != nil {
		fmt.Print(err)
		os.Exit(-1)
	}

	releaseExists := false
	releaseId, uploadUrl, releaseUrl, targetCommitSha, err := getReleaseInfo(release)
	if err == nil {
		releaseExists = true
		if info.commit != targetCommitSha {
			fmt.Printf("Found existing release but its commit SHA doesn't match the current one: %s vs %s\n", info.commit, targetCommitSha)
			fmt.Printf("Deleting the existing release to recreate it with the current commit SHA %s\n", info.commit)

			err = deleteRelease(ctx, client, info, release)
			if err != nil {
				fmt.Print(err)
				os.Exit(-1)
			}

			releaseExists = false

			if info.isPreRelease {
				fmt.Printf("Since the existing release was pre-release one, need to also remove the tag corresponding to it")
				err = deleteTag(tokenizedClient, info)
				if err != nil {
					fmt.Print(err)
					os.Exit(-1)
				}
			}
		}
	}

	if !releaseExists {
		release, err = createRelease(client, info, pReleaseBody)
		if err != nil {
			fmt.Print(err)
			os.Exit(-1)
		}

		releaseId, uploadUrl, releaseUrl, targetCommitSha, err = getReleaseInfo(release)
		if err != nil {
			fmt.Print(err)
			os.Exit(-1)
		}
	}

	files = commandLineFiles(flag.Args())
	err = uploadBinaries(files, uploadUrl, ctx, client, release, info)
	if err != nil {
		fmt.Print(err)
		os.Exit(-1)
	}

	return
}

func commandLineFiles(files []string) []string {
	if runtime.GOOS == "windows" {
		args := make([]string, 0, len(files))
		for _, name := range files {
			if matches, err := filepath.Glob(name); err != nil {
				args = append(args, name) // Invalid pattern
			} else if matches != nil { // At least one match
				args = append(args, matches...)
			}
		}
		return args
	}
	return files
}

type buildEventInfo struct {
	token          string
	tag            string
	commit         string
	branch         string
	repo           string
	owner          string
	isPullRequest  bool
	releaseTagName string
	releaseTitle   string
	isPreRelease   bool
	isTravisCi     bool
	buildId        string
}

func getBuildEventInfo(pSuffix *string) (*buildEventInfo, error) {
	// Check whether the app is run during Travis CI or AppVeyor CI build
	appVeyorEnvVar := os.Getenv("APPVEYOR")
	travisCiEnvVar := os.Getenv("TRAVIS")
	isTravisCi := travisCiEnvVar == "true"
	isAppVeyor := appVeyorEnvVar == "True"
	if !isTravisCi && !isAppVeyor {
		fmt.Println("Neither Travis CI build nor AppVeyor build. Not doing anything")
		return nil, nil
	}

	// Get GitHub API token from the environment variable
	var token string
	if isAppVeyor {
		token = os.Getenv("auth_token")
	} else {
		token = os.Getenv("GITHUB_TOKEN")
	}

	if token == "" {
		return nil, errors.New("No GitHub access token, can't proceed")
	}

	// Get various build information from environment variables
	// specific to Travis CI and AppVeyor CI
	var info buildEventInfo
	info.isTravisCi = isTravisCi

	repoSlug := ""

	if isAppVeyor {
		fmt.Println("Running on Travis CI")
		info.branch = os.Getenv("APPVEYOR_REPO_BRANCH")
		info.tag = os.Getenv("APPVEYOR_REPO_TAG_NAME")
		info.commit = os.Getenv("APPVEYOR_REPO_COMMIT")
		repoSlug = os.Getenv("APPVEYOR_REPO_NAME")
		info.buildId = os.Getenv("APPVEYOR_JOB_ID")
		info.isPullRequest = os.Getenv("APPVEYOR_PULL_REQUEST_NUMBER") != ""
	} else {
		fmt.Println("Running on AppVeyor CI")
		info.branch = os.Getenv("TRAVIS_BRANCH")
		info.tag = os.Getenv("TRAVIS_TAG")
		info.commit = os.Getenv("TRAVIS_COMMIT")
		repoSlug = os.Getenv("TRAVIS_REPO_SLUG")
		info.buildId = os.Getenv("TRAVIS_JOB_ID")
		info.isPullRequest = os.Getenv("TRAVIS_EVENT_TYPE") == "pull_request"
	}

	if info.isPullRequest {
		fmt.Println("Current build is the one triggered by a pull request, won't do anything")
		return nil, nil
	}

	if info.branch == "" {
		info.branch = "master"
	}

	fmt.Println("Commit: ", info.commit)

	repoSlugSplitted := strings.Split(repoSlug, "/")
	if len(repoSlugSplitted) != 2 {
		fmt.Printf("Error splitting APPVEYOR_REPO_NAME into owner and repo: %s\n", repoSlug)
		os.Exit(-1)
	}

	info.owner = repoSlugSplitted[0]
	info.repo = repoSlugSplitted[1]

	var osName string
	if runtime.GOOS == "darwin" {
		osName = "mac"
	} else {
		osName = runtime.GOOS
	}

	if pSuffix == nil || *pSuffix == tag {
		info.releaseTagName = tag
		info.releaseTitle = "Release build (" + tag + ")"
	} else if pSuffix != nil {
		suffix := osName + "-" + *pSuffix
		info.releaseTagName = "continuous-" + suffix
		info.releaseTitle = "Continuous build (" + suffix + ")"
		info.isPreRelease = true
	} else {
		info.releaseTagName = "continuous-" + osName // Do not use "latest" as it is reserved by GitHub
		info.releaseTitle = "Continuous build (" + osName + ")"
		info.isPreRelease = true
	}

	return &info, nil
}

func checkResponse(response *github.Response) error {
	if response == nil {
		return errors.New("Response is nil")
	}

	if response.Response == nil {
		return errors.New("Failed to fetch tag information: no HTTP response")
	}

	if response.Response.StatusCode != 200 {
		return fmt.Errorf("Failed to fetch tag information: bad status code %d: %s\n", response.Response.StatusCode, response.Response.Status)
	}

	return nil
}

func getReleaseInfo(release *github.RepositoryRelease) (id int, uploadUrl, releaseUrl, targetCommitSha string, err error) {
	if release == nil {
		err = fmt.Errorf("Release is nil")
		return
	}

	if release.ID == nil {
		err = fmt.Errorf("Release's ID is nil")
		return
	}
	id = release.GetID()

	uploadUrl = release.GetUploadURL()
	if uploadUrl == "" {
		err = fmt.Errorf("Release's upload URL is empty")
		return
	}

	releaseUrl = release.GetURL()
	if releaseUrl == "" {
		err = fmt.Errorf("Release's URL is empty")
		return
	}

	targetCommitSha = release.GetTargetCommitish()
	if targetCommitSha == "" {
		err = fmt.Errorf("Release's target commit SHA is empty")
		return
	}

	return
}

func deleteRelease(ctx *context.Context, client *github.Client, info *buildEventInfo, release *github.RepositoryRelease) error {
	response, err := client.Repositories.DeleteRelease(ctx, info.owner, info.repo, release.GetID())
	if err != nil {
		return err
	}

	if response != nil {
		defer response.Body.Close()
	}

	err = checkResponse(response)
	if err != nil {
		return err
	}

	return nil
}

func createRelease(ctx *context.Context, client *github.Client, info *buildEventInfo, pReleaseBody *string) (*github.RepositoryRelease, error) {
	release := new(github.RepositoryRelease)
	release.TagName = new(string)
	*release.TagName = info.releaseTagName
	release.TargetCommitish = new(string)
	*release.TargetCommitish = info.commit
	release.Name = new(string)
	*release.Name = info.releaseTitle

	release.Body = new(string)
	if pReleaseBody == nil {
		if info.isTravisCi && info.buildId != "" {
			*pReleaseBody = "Travis CI build log: https://travis-ci.org/" + info.owner + "/" + info.repo + "/builds/" + info.buildId + "/"
		} else if !info.isTravisCi && info.buildId != "" {
			*pReleaseBody = "AppVeyor CI build log: https://ci.appveyor.com/api/buildjobs/" + info.buildId + "/log"
		}
	} else {
		*release.Body = *pReleaseBody
	}

	release, response, err := client.Repositories.CreateRelease(ctx, info.owner, info.repo, release)
	if err != nil {
		return err
	}

	if response != nil {
		defer response.Body.Close()
	}

	err = checkResponse(response)
	if err != nil {
		return err
	}

	return nil, nil
}

func deleteTag(client *http.Client, info *buildEventInfo) error {
	// GitHub guys haven't really created any actual API for tag deletion so need to do it the hard way
	deleteUrl = "https://api.github.com/repos/" + info.owner + "/" + info.repo + "/git/refs/tags/" + info.releaseTagName
	req, err := client.NewRequest("DELETE", deleteUrl)
	if err != nil {
		return err
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	if response != nil {
		defer response.Body.Close()
	}

	err = checkResponse(response)
	if err != nil {
		return err
	}

	return nil
}

func uploadBinaries(filenames []string, uploadUrl string, context *context.Context, client *github.Client,
	release *github.RepositoryRelease, info *buildEventInfo) error {
	for _, filename := range filenames {
		file, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer file.Close()

		var options github.UploadOptions
		options.Name = filepath.Base(filename)

		_, response, err := client.Repositories.UploadReleaseAsset(ctx, info.owner, info.repo, release.GetID(), file)
		if err != nil {
			return err
		}

		if response != nil {
			defer response.Body.Close()
		}

		err = checkResponse(response)
		if err != nil {
			return err
		}
	}

	return nil
}
