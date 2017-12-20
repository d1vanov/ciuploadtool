package uploader

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func init() {
	rand.Seed(time.Now().Unix())
}

func TestNewReleaseWithSingleUploadedBinary(t *testing.T) {
	file, err := ioutil.TempFile("", "singleUploadedBinary.txt")
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the single uploaded binary: %v", err)
	}

	commit := generateRandomString(16)
	branch := "master"
	tag := "continuous-master"
	repoSlug := "d1vanov/ciuploadtool"
	isPullRequest := false

	releaseSuffix := "master"
	releaseBody := ""

	for i := 0; i < 2; i++ {
		if i == 0 {
			setupTravisCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "Travis CI build log: https://travis-ci.org/d1vanov/ciuploadtool/builds/" + os.Getenv("TRAVIS_JOB_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: https://ci.appveyor.com/api/buildjobs/" + os.Getenv("APPVEYOR_JOB_ID") + "/log"
		}

		client, err := uploadImpl(clientFactoryFunc(newTstClient), releaseFactoryFunc(newTstRelease), []string{file.Name()},
			releaseSuffix, releaseBody)
		if err != nil {
			t.Errorf("Failed to upload the single binary: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Errorf("Uploading single binary to new release failed: no releases within the returned client")
		}

		release := tstClient.releases[0]
		assets := release.GetAssets()
		if len(assets) != 1 {
			t.Errorf("Uploading single binary to new release failed: no assets within the release")
		}
	}
}

func setupTravisCiEnvVars(commit string, branch string, tag string, repoSlug string, isPullRequest bool) {
	os.Unsetenv("APPVEYOR")
	os.Setenv("TRAVIS", "true")
	os.Setenv("GITHUB_TOKEN", "fake_token")
	os.Setenv("TRAVIS_BRANCH", branch)
	os.Setenv("TRAVIS_TAG", tag)
	os.Setenv("TRAVIS_COMMIT", commit)
	os.Setenv("TRAVIS_REPO_SLUG", repoSlug)
	os.Setenv("TRAVIS_JOB_ID", generateRandomString(10))
	if isPullRequest {
		os.Setenv("TRAVIS_EVENT_TYPE", "pull_request")
	} else {
		os.Setenv("TRAVIS_EVENT_TYPE", "non_pull_request")
	}
}

func setupAppVeyorCiEnvVars(commit string, branch string, tag string, repoSlug string, isPullRequest bool) {
	os.Unsetenv("TRAVIS")
	os.Setenv("APPVEYOR", "True")
	os.Setenv("auth_token", "fake_token")
	os.Setenv("APPVEYOR_REPO_BRANCH", branch)
	os.Setenv("APPVEYOR_REPO_TAG_NAME", tag)
	os.Setenv("APPVEYOR_REPO_COMMIT", commit)
	os.Setenv("APPVEYOR_REPO_NAME", repoSlug)
	os.Setenv("APPVEYOR_JOB_ID", generateRandomString(10))
	if isPullRequest {
		os.Setenv("APPVEYOR_PULL_REQUEST_NUMBER", generateRandomString(5))
	} else {
		os.Unsetenv("APPVEYOR_PULL_REQUEST_NUMBER")
	}
}

func generateRandomString(numChars int) string {
	b := make([]rune, numChars)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
