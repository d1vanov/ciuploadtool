package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	var pSuffix = flag.String("suffix", "", "Optional suffix for the names of created continuous releases")
	var pReleaseBody = flag.String("relbody", "", "Optional content for the body of created releases")
	flag.Parse()

	if flag.NArg() < 1 {
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
	ctx := context.Background()
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: info.token})
	tokenizedClient := oauth2.NewClient(ctx, tokenSource)

	client := github.NewClient(tokenizedClient)

	// Fetch release info (if release with target tag name already exists)
	releaseExists := false

	release, response, err := client.Repositories.GetReleaseByTag(ctx, info.owner, info.repo, info.releaseTagName)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == 404 {
			err = nil
		}
		if err != nil {
			fmt.Printf("Failed to fetch release information: %v", err)
			os.Exit(-1)
		}
	} else {
		err = checkResponse(response)
		if err != nil {
			fmt.Print(err)
			os.Exit(-1)
		}
		releaseExists = true
	}

	if releaseExists {
		targetCommitSha, err := getReleaseTargetCommitSha(release)
		if err == nil {
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
	}

	if !releaseExists {
		release, err = createRelease(ctx, client, info, pReleaseBody)
		if err != nil {
			fmt.Print(err)
			os.Exit(-1)
		}
	}

	files := commandLineFiles(flag.Args())
	err = uploadBinaries(files, ctx, client, release, info)
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

	var info buildEventInfo

	// Get GitHub API token from the environment variable
	if isAppVeyor {
		info.token = os.Getenv("auth_token")
	} else {
		info.token = os.Getenv("GITHUB_TOKEN")
	}

	if info.token == "" {
		return nil, errors.New("No GitHub access token, can't proceed")
	}

	// Get various build information from environment variables
	// specific to Travis CI and AppVeyor CI
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

	if pSuffix == nil || *pSuffix == info.tag {
		info.releaseTagName = info.tag
		info.releaseTitle = "Release build (" + info.tag + ")"
	} else if pSuffix != nil {
		suffix := osName + "-" + *pSuffix
		info.releaseTagName = "continuous-" + suffix
		info.releaseTitle = "Continuous build (" + info.releaseTagName + ")"
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
		return errors.New("No HTTP response")
	}

	if response.Response.StatusCode < 200 || response.Response.StatusCode > 299 {
		return fmt.Errorf("Bad status code %d: %s\n", response.Response.StatusCode, response.Response.Status)
	}

	return nil
}

func getReleaseTargetCommitSha(release *github.RepositoryRelease) (targetCommitSha string, err error) {
	if release == nil {
		err = fmt.Errorf("Release is nil")
		return
	}

	targetCommitSha = release.GetTargetCommitish()
	if targetCommitSha == "" {
		err = fmt.Errorf("Release's target commit SHA is empty")
		return
	}

	return
}

func deleteRelease(ctx context.Context, client *github.Client, info *buildEventInfo, release *github.RepositoryRelease) error {
	response, err := client.Repositories.DeleteRelease(ctx, info.owner, info.repo, release.GetID())
	if err != nil {
		return err
	}

	if response != nil {
		defer response.Body.Close()
	}

	err = checkResponse(response)
	if err != nil {
		return fmt.Errorf("Failed to delete release: %v", err)
	}

	return nil
}

func createRelease(ctx context.Context, client *github.Client, info *buildEventInfo, pReleaseBody *string) (*github.RepositoryRelease, error) {
	release := new(github.RepositoryRelease)
	release.TagName = new(string)
	*release.TagName = info.releaseTagName
	release.TargetCommitish = new(string)
	*release.TargetCommitish = info.commit
	release.Name = new(string)
	*release.Name = info.releaseTitle
	release.Prerelease = new(bool)
	*release.Prerelease = info.isPreRelease

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
		return nil, err
	}

	if response != nil {
		defer response.Body.Close()
	}

	err = checkResponse(response)
	if err != nil {
		return nil, fmt.Errorf("Failed to create release: %v", err)
	}

	return release, nil
}

func deleteTag(client *http.Client, info *buildEventInfo) error {
	// GitHub guys haven't really created any actual API for tag deletion so need to do it the hard way
	deleteUrl := "https://api.github.com/repos/" + info.owner + "/" + info.repo + "/git/refs/tags/" + info.releaseTagName
	request, err := http.NewRequest("DELETE", deleteUrl, nil)
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

	if response.StatusCode != 200 {
		return fmt.Errorf("Failed to fetch response to tag deletion: status code = %d: %s", response.StatusCode, response.Status)
	}

	return nil
}

func uploadBinaries(filenames []string, ctx context.Context, client *github.Client,
	release *github.RepositoryRelease, info *buildEventInfo) error {
	for _, filename := range filenames {
		file, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			return err
		}

		mode := stat.Mode()
		if !mode.IsRegular() {
			fmt.Printf("Skipping dir %s\n", filename)
			continue
		}

		fmt.Printf("Trying to upload file: %s\n", filename)

		var options github.UploadOptions
		options.Name = filepath.Base(filename)

		_, response, err := client.Repositories.UploadReleaseAsset(ctx, info.owner, info.repo, release.GetID(), &options, file)
		if err != nil {
			return err
		}

		if response != nil {
			defer response.Body.Close()
		}

		err = checkResponse(response)
		if err != nil {
			return fmt.Errorf("Bad response on attempt to upload release asset: %v", err)
		}
	}

	return nil
}
