package uploader

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func init() {
	rand.Seed(time.Now().Unix())
}

func TestNewReleaseWithSingleUploadedBinary(t *testing.T) {
	binaryContent := "Binary content"
	file, err := setupSampleAssetFile("singleUploadedBinary.txt", binaryContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the single "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(file.Name())
	defer file.Close()

	commit := generateRandomString(16)
	branch := "master"
	tag := "continuous-master"
	owner := "d1vanov"
	repo := "ciuploadtool"
	repoSlug := owner + "/" + repo
	isPullRequest := false

	releaseSuffix := "master"
	releaseBody := ""

	for i := 0; i < 2; i++ {
		if i == 0 {
			setupTravisCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "Travis CI build log: " +
				"https://travis-ci.org/d1vanov/ciuploadtool/builds/" +
				os.Getenv("TRAVIS_BUILD_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: " +
				"https://ci.appveyor.com/project/" + owner + "/" + repo +
				"/build/" + os.Getenv("APPVEYOR_BUILD_VERSION")
		}

		client, err := uploadImpl(
			clientFactoryFunc(newTstClient),
			releaseFactoryFunc(newTstRelease),
			[]string{file.Name()},
			releaseSuffix,
			releaseBody,
			false)
		if err != nil {
			t.Fatalf("Failed to upload the single binary: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Fatalf("Uploading single binary to new release failed: no releases " +
				"within the returned client")
		}

		release := tstClient.releases[0]
		assets := release.GetAssets()
		if len(assets) != 1 {
			t.Fatalf("Uploading single binary to new release failed: no assets " +
				"within the release")
		}

		asset := assets[0]
		tstAsset, ok := asset.(TstReleaseAsset)
		if !ok {
			t.Fatalf(
				"Failed to cast the release asset to TstReleaseAsset: %v",
				err)
		}

		tstAssetContent := tstAsset.GetContent()
		if tstAssetContent != binaryContent {
			t.Fatalf("The contents of uploaded release asset don't match " +
				"the original resource file's contents")
		}
	}
}

func TestNewReleaseWithSeveralUploadedBinaries(t *testing.T) {
	firstFileContent := "First file content"
	firstFile, err := setupSampleAssetFile(
		"firstUploadedBinary.txt",
		firstFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the first "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(firstFile.Name())
	defer firstFile.Close()

	secondFileContent := "Second file content"
	secondFile, err := setupSampleAssetFile(
		"secondUploadedBinary.txt",
		secondFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the second "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(secondFile.Name())
	defer secondFile.Close()

	thirdFileContent := "Third file content"
	thirdFile, err := setupSampleAssetFile(
		"thirdUploadedBinary.txt",
		thirdFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the third "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(thirdFile.Name())
	defer thirdFile.Close()

	commit := generateRandomString(16)
	branch := "master"
	tag := "continuous-master"
	owner := "d1vanov"
	repo := "ciuploadtool"
	repoSlug := owner + "/" + repo
	isPullRequest := false

	releaseSuffix := "master"
	releaseBody := ""

	filenames := make([]string, 0, 3)
	filenames = append(filenames, firstFile.Name())
	filenames = append(filenames, secondFile.Name())
	filenames = append(filenames, thirdFile.Name())

	for i := 0; i < 2; i++ {
		if i == 0 {
			setupTravisCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "Travis CI build log: " +
				"https://travis-ci.org/d1vanov/ciuploadtool/builds/" +
				os.Getenv("TRAVIS_BUILD_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: " +
				"https://ci.appveyor.com/project/" + owner + "/" + repo +
				"/build/" + os.Getenv("APPVEYOR_BUILD_VERSION")
		}

		client, err := uploadImpl(
			clientFactoryFunc(newTstClient),
			releaseFactoryFunc(newTstRelease),
			filenames,
			releaseSuffix,
			releaseBody,
			false)
		if err != nil {
			t.Fatalf("Failed to upload one of binaries: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Fatalf("Uploading one of binaries to new release failed: no " +
				"releases within the returned client")
		}

		release := tstClient.releases[0]
		assets := release.GetAssets()
		if len(assets) != 3 {
			t.Fatalf("Uploading one of binaries to new release failed: wrong " +
				"number of assets within the release")
		}

		for _, asset := range assets {
			tstAsset, ok := asset.(TstReleaseAsset)
			if !ok {
				t.Fatalf(
					"Failed to cast the release asset to TstReleaseAsset: %v",
					err)
			}

			tstAssetContent := tstAsset.GetContent()

			if tstAsset.GetName() == filepath.Base(firstFile.Name()) {
				if tstAssetContent != firstFileContent {
					t.Fatalf("The contents of the first uploaded release asset " +
						"don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == filepath.Base(secondFile.Name()) {
				if tstAssetContent != secondFileContent {
					t.Fatalf("The contents of the second uploaded release asset " +
						"don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == filepath.Base(thirdFile.Name()) {
				if tstAssetContent != thirdFileContent {
					t.Fatalf("The contents of the third uploaded release asset " +
						"don't match the original resource file's contents")
				}
			} else {
				t.Fatalf("Found unidentified release asset: %+v", tstAsset)
			}
		}
	}
}

func TestInitiallyEmptyExistingReleaseWithSingleUploadedBinary(t *testing.T) {
	binaryContent := "Binary content"
	file, err := setupSampleAssetFile("singleUploadedBinary.txt", binaryContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the single "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(file.Name())
	defer file.Close()

	commit := generateRandomString(16)
	branch := "master"
	tag := "continuous-master"
	owner := "d1vanov"
	repo := "ciuploadtool"
	repoSlug := owner + "/" + repo
	isPullRequest := false

	releaseSuffix := "master"
	releaseBody := "Continuous release"

	for i := 0; i < 2; i++ {
		if i == 0 {
			setupTravisCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "Travis CI build log: " +
				"https://travis-ci.org/d1vanov/ciuploadtool/builds/" +
				os.Getenv("TRAVIS_BUILD_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: " +
				"https://ci.appveyor.com/project/" + owner + "/" + repo +
				"/build/" + os.Getenv("APPVEYOR_BUILD_VERSION")
		}

		clientFactory := func(
			gitHubToken string,
			owner string,
			repo string) Client {

			info, err := collectBuildEventInfo(releaseSuffix, false)
			if err != nil {
				panic(err)
			}
			tstRelease := newTstRelease(releaseBody, info, false).(*TstRelease)
			tstClient := newTstClient(gitHubToken, owner, repo).(*TstClient)
			tstClient.releases = append(tstClient.releases, *tstRelease)
			return tstClient
		}

		client, err := uploadImpl(
			clientFactoryFunc(clientFactory),
			releaseFactoryFunc(newTstRelease),
			[]string{file.Name()},
			releaseSuffix,
			releaseBody,
			false)
		if err != nil {
			t.Fatalf("Failed to upload one of binaries: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Fatalf("Uploading one of binaries to existing release failed: no " +
				"releases within the returned client")
		}

		release := tstClient.releases[0]
		assets := release.GetAssets()
		if len(assets) != 1 {
			t.Fatalf("Uploading one of binaries to existing release failed: " +
				"no assets within the release")
		}

		asset := assets[0]
		tstAsset, ok := asset.(TstReleaseAsset)
		if !ok {
			t.Fatalf(
				"Failed to cast the release asset to TstReleaseAsset: %v",
				err)
		}

		tstAssetContent := tstAsset.GetContent()
		if tstAssetContent != binaryContent {
			t.Fatalf("The contents of uploaded release asset don't match " +
				"the original resource file's contents")
		}
	}
}

func TestInitiallyEmptyExistingReleaseWithSeveralUploadedBinaries(t *testing.T) {
	firstFileContent := "First file content"
	firstFile, err := setupSampleAssetFile(
		"firstUploadedBinary.txt",
		firstFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the first "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(firstFile.Name())
	defer firstFile.Close()

	secondFileContent := "Second file content"
	secondFile, err := setupSampleAssetFile(
		"secondUploadedBinary.txt",
		secondFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the second "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(secondFile.Name())
	defer secondFile.Close()

	thirdFileContent := "Third file content"
	thirdFile, err := setupSampleAssetFile(
		"thirdUploadedBinary.txt",
		thirdFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the third "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(thirdFile.Name())
	defer thirdFile.Close()

	commit := generateRandomString(16)
	branch := "master"
	tag := "continuous-master"
	owner := "d1vanov"
	repo := "ciuploadtool"
	repoSlug := owner + "/" + repo
	isPullRequest := false

	releaseSuffix := "master"
	releaseBody := ""

	filenames := make([]string, 0, 3)
	filenames = append(filenames, firstFile.Name())
	filenames = append(filenames, secondFile.Name())
	filenames = append(filenames, thirdFile.Name())

	for i := 0; i < 2; i++ {
		if i == 0 {
			setupTravisCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "Travis CI build log: " +
				"https://travis-ci.org/d1vanov/ciuploadtool/builds/" +
				os.Getenv("TRAVIS_BUILD_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: " +
				"https://ci.appveyor.com/project/" + owner + "/" + repo +
				"/build/" + os.Getenv("APPVEYOR_BUILD_VERSION")
		}

		clientFactory := func(
			gitHubToken string,
			owner string,
			repo string) Client {

			info, err := collectBuildEventInfo(releaseSuffix, false)
			if err != nil {
				panic(err)
			}
			tstRelease := newTstRelease(releaseBody, info, false).(*TstRelease)
			tstClient := newTstClient(gitHubToken, owner, repo).(*TstClient)
			tstClient.releases = append(tstClient.releases, *tstRelease)
			return tstClient
		}

		client, err := uploadImpl(
			clientFactoryFunc(clientFactory),
			releaseFactoryFunc(newTstRelease),
			filenames,
			releaseSuffix,
			releaseBody,
			false)
		if err != nil {
			t.Fatalf("Failed to upload one of binaries: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Fatalf("Uploading one of binaries to existing release failed: " +
				"no releases within the returned client")
		}

		release := tstClient.releases[0]
		assets := release.GetAssets()
		if len(assets) != 3 {
			t.Fatalf("Uploading one of binaries to existing release failed: " +
				"wrong number of assets within the release")
		}

		for _, asset := range assets {
			tstAsset, ok := asset.(TstReleaseAsset)
			if !ok {
				t.Fatalf(
					"Failed to cast the release asset to TstReleaseAsset: %v",
					err)
			}

			tstAssetContent := tstAsset.GetContent()

			if tstAsset.GetName() == filepath.Base(firstFile.Name()) {
				if tstAssetContent != firstFileContent {
					t.Fatalf("The contents of the first uploaded release asset " +
						"don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == filepath.Base(secondFile.Name()) {
				if tstAssetContent != secondFileContent {
					t.Fatalf("The contents of the second uploaded release asset " +
						"don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == filepath.Base(thirdFile.Name()) {
				if tstAssetContent != thirdFileContent {
					t.Fatalf("The contents of the third uploaded release asset " +
						"don't match the original resource file's contents")
				}
			} else {
				t.Fatalf("Found unidentified release asset: %+v", tstAsset)
			}
		}
	}
}

func TestExistingReleaseWithSingleUploadedBinary(t *testing.T) {
	binaryContent := "Binary content"
	file, err := setupSampleAssetFile("singleUploadedBinary.txt", binaryContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the single "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(file.Name())
	defer file.Close()

	commit := generateRandomString(16)
	branch := "master"
	tag := "continuous-master"
	owner := "d1vanov"
	repo := "ciuploadtool"
	repoSlug := owner + "/" + repo
	isPullRequest := false

	releaseSuffix := "master"
	releaseBody := "Continuous release"

	for i := 0; i < 2; i++ {
		if i == 0 {
			setupTravisCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "Travis CI build log: " +
				"https://travis-ci.org/d1vanov/ciuploadtool/builds/" +
				os.Getenv("TRAVIS_BUILD_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: " +
				"https://ci.appveyor.com/project/" + owner + "/" + repo +
				"/build/" + os.Getenv("APPVEYOR_BUILD_VERSION")
		}

		clientFactory := func(
			gitHubToken string,
			owner string,
			repo string) Client {

			info, err := collectBuildEventInfo(releaseSuffix, false)
			if err != nil {
				panic(err)
			}
			tstRelease := newTstRelease(releaseBody, info, false).(*TstRelease)
			tstAsset := TstReleaseAsset{
				id:      lastFreeReleaseAssetId,
				name:    filepath.Base(file.Name()),
				content: binaryContent,
			}
			tstRelease.assets = append(tstRelease.assets, tstAsset)
			tstClient := newTstClient(gitHubToken, owner, repo).(*TstClient)
			tstClient.releases = append(tstClient.releases, *tstRelease)
			return tstClient
		}

		client, err := uploadImpl(
			clientFactoryFunc(clientFactory),
			releaseFactoryFunc(newTstRelease),
			[]string{file.Name()},
			releaseSuffix,
			releaseBody,
			false)
		if err != nil {
			t.Fatalf("Failed to upload the single binary: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Fatalf("Uploading single binary to existing release failed: " +
				"no releases within the returned client")
		}

		release := tstClient.releases[0]
		assets := release.GetAssets()
		if len(assets) != 1 {
			t.Fatalf("Uploading single binary to existing release failed: " +
				"no assets within the release")
		}

		asset := assets[0]
		tstAsset, ok := asset.(TstReleaseAsset)
		if !ok {
			t.Fatalf(
				"Failed to cast the release asset to TstReleaseAsset: %v",
				err)
		}

		tstAssetContent := tstAsset.GetContent()
		if tstAssetContent != binaryContent {
			t.Fatalf("The contents of uploaded release asset don't match " +
				"the original resource file's contents")
		}
	}
}

func TestExistingReleaseWithSeveralUploadedBinariesAllBeingReplacements(
	t *testing.T) {

	firstFileContent := "First file content"
	firstFile, err := setupSampleAssetFile(
		"firstUploadedBinary.txt",
		firstFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the first "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(firstFile.Name())
	defer firstFile.Close()

	secondFileContent := "Second file content"
	secondFile, err := setupSampleAssetFile(
		"secondUploadedBinary.txt",
		secondFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the second "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(secondFile.Name())
	defer secondFile.Close()

	thirdFileContent := "Third file content"
	thirdFile, err := setupSampleAssetFile(
		"thirdUploadedBinary.txt",
		thirdFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the third "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(thirdFile.Name())
	defer thirdFile.Close()

	commit := generateRandomString(16)
	branch := "master"
	tag := "continuous-master"
	owner := "d1vanov"
	repo := "ciuploadtool"
	repoSlug := owner + "/" + repo
	isPullRequest := false

	releaseSuffix := "master"
	releaseBody := ""

	filenames := make([]string, 0, 3)
	filenames = append(filenames, firstFile.Name())
	filenames = append(filenames, secondFile.Name())
	filenames = append(filenames, thirdFile.Name())

	for i := 0; i < 2; i++ {
		if i == 0 {
			setupTravisCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "Travis CI build log: " +
				"https://travis-ci.org/d1vanov/ciuploadtool/builds/" +
				os.Getenv("TRAVIS_BUILD_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: " +
				"https://ci.appveyor.com/project/" + owner + "/" + repo +
				"/build/" + os.Getenv("APPVEYOR_BUILD_VERSION")
		}

		clientFactory := func(
			gitHubToken string,
			owner string,
			repo string) Client {

			info, err := collectBuildEventInfo(releaseSuffix, false)
			if err != nil {
				panic(err)
			}
			tstRelease := newTstRelease(releaseBody, info, false).(*TstRelease)
			firstAsset := TstReleaseAsset{
				id:      lastFreeReleaseAssetId,
				name:    filepath.Base(firstFile.Name()),
				content: firstFileContent,
			}
			lastFreeReleaseAssetId++
			secondAsset := TstReleaseAsset{
				id:      lastFreeReleaseAssetId,
				name:    filepath.Base(secondFile.Name()),
				content: secondFileContent,
			}
			lastFreeReleaseAssetId++
			thirdAsset := TstReleaseAsset{
				id:      lastFreeReleaseAssetId,
				name:    filepath.Base(thirdFile.Name()),
				content: thirdFileContent,
			}
			lastFreeReleaseAssetId++
			tstRelease.assets = append(tstRelease.assets, firstAsset)
			tstRelease.assets = append(tstRelease.assets, secondAsset)
			tstRelease.assets = append(tstRelease.assets, thirdAsset)
			tstClient := newTstClient(gitHubToken, owner, repo).(*TstClient)
			tstClient.releases = append(tstClient.releases, *tstRelease)
			return tstClient
		}

		client, err := uploadImpl(
			clientFactoryFunc(clientFactory),
			releaseFactoryFunc(newTstRelease),
			filenames,
			releaseSuffix,
			releaseBody,
			false)
		if err != nil {
			t.Fatalf("Failed to upload one of binaries: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Fatalf("Uploading one of binaries to existing release failed: " +
				"no releases within the returned client")
		}

		release := tstClient.releases[0]
		assets := release.GetAssets()
		if len(assets) != 3 {
			t.Fatalf("Uploading one of binaries to existing release failed: " +
				"no assets within the release")
		}

		for _, asset := range assets {
			tstAsset, ok := asset.(TstReleaseAsset)
			if !ok {
				t.Fatalf(
					"Failed to cast the release asset to TstReleaseAsset: %v",
					err)
			}

			tstAssetContent := tstAsset.GetContent()

			if tstAsset.GetName() == filepath.Base(firstFile.Name()) {
				if tstAssetContent != firstFileContent {
					t.Fatalf("The contents of the first uploaded release asset " +
						"don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == filepath.Base(secondFile.Name()) {
				if tstAssetContent != secondFileContent {
					t.Fatalf("The contents of the second uploaded release asset " +
						"don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == filepath.Base(thirdFile.Name()) {
				if tstAssetContent != thirdFileContent {
					t.Fatalf("The contents of the third uploaded release asset " +
						"don't match the original resource file's contents")
				}
			} else {
				t.Fatalf("Found unidentified release asset: %+v", tstAsset)
			}
		}
	}
}

func TestExistingReleaseWithSeveralUploadedBinariesNotAllBeingReplacements(
	t *testing.T) {

	firstFileContent := "First file content"
	firstFile, err := setupSampleAssetFile(
		"firstUploadedBinary.txt",
		firstFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the first "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(firstFile.Name())
	defer firstFile.Close()

	secondFileContent := "Second file content"
	secondFile, err := setupSampleAssetFile(
		"secondUploadedBinary.txt",
		secondFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the second "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(secondFile.Name())
	defer secondFile.Close()

	thirdFileContent := "Third file content"
	thirdFile, err := setupSampleAssetFile(
		"thirdUploadedBinary.txt",
		thirdFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the third "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(thirdFile.Name())
	defer thirdFile.Close()

	fourthAssetContent := "Fourth file content"
	fourthAssetName := "fourthUploadedBinary.txt"

	fifthAssetContent := "Fifth file content"
	fifthAssetName := "fifthUploadedBinary.txt"

	commit := generateRandomString(16)
	branch := "master"
	tag := "continuous-master"
	owner := "d1vanov"
	repo := "ciuploadtool"
	repoSlug := owner + "/" + repo
	isPullRequest := false

	releaseSuffix := "master"
	releaseBody := ""

	filenames := make([]string, 0, 3)
	filenames = append(filenames, firstFile.Name())
	filenames = append(filenames, secondFile.Name())
	filenames = append(filenames, thirdFile.Name())

	for i := 0; i < 2; i++ {
		if i == 0 {
			setupTravisCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "Travis CI build log: " +
				"https://travis-ci.org/d1vanov/ciuploadtool/builds/" +
				os.Getenv("TRAVIS_BUILD_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: " +
				"https://ci.appveyor.com/project/" + owner + "/" + repo +
				"/build/" + os.Getenv("APPVEYOR_BUILD_VERSION")
		}

		clientFactory := func(
			gitHubToken string,
			owner string,
			repo string) Client {

			info, err := collectBuildEventInfo(releaseSuffix, false)
			if err != nil {
				panic(err)
			}
			tstRelease := newTstRelease(releaseBody, info, false).(*TstRelease)
			firstAsset := TstReleaseAsset{
				id:      lastFreeReleaseAssetId,
				name:    filepath.Base(firstFile.Name()),
				content: firstFileContent,
			}
			lastFreeReleaseAssetId++
			secondAsset := TstReleaseAsset{
				id:      lastFreeReleaseAssetId,
				name:    filepath.Base(secondFile.Name()),
				content: secondFileContent,
			}
			lastFreeReleaseAssetId++
			thirdAsset := TstReleaseAsset{
				id:      lastFreeReleaseAssetId,
				name:    filepath.Base(thirdFile.Name()),
				content: thirdFileContent,
			}
			lastFreeReleaseAssetId++
			fourthAsset := TstReleaseAsset{
				id:      lastFreeReleaseAssetId,
				name:    fourthAssetName,
				content: fourthAssetContent,
			}
			lastFreeReleaseAssetId++
			fifthAsset := TstReleaseAsset{
				id:      lastFreeReleaseAssetId,
				name:    fifthAssetName,
				content: fifthAssetContent,
			}
			lastFreeReleaseAssetId++
			tstRelease.assets = append(tstRelease.assets, firstAsset)
			tstRelease.assets = append(tstRelease.assets, secondAsset)
			tstRelease.assets = append(tstRelease.assets, thirdAsset)
			tstRelease.assets = append(tstRelease.assets, fourthAsset)
			tstRelease.assets = append(tstRelease.assets, fifthAsset)
			tstClient := newTstClient(gitHubToken, owner, repo).(*TstClient)
			tstClient.releases = append(tstClient.releases, *tstRelease)
			return tstClient
		}

		client, err := uploadImpl(
			clientFactoryFunc(clientFactory),
			releaseFactoryFunc(newTstRelease),
			filenames,
			releaseSuffix,
			releaseBody,
			false)
		if err != nil {
			t.Fatalf("Failed to upload one of binaries: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Fatalf("Uploading one of binaries to existing release failed: " +
				"no releases within the returned client")
		}

		release := tstClient.releases[0]
		assets := release.GetAssets()
		if len(assets) != 5 {
			t.Fatalf("Uploading one of binaries to existing release failed: " +
				"wrong number of assets within the release")
		}

		for _, asset := range assets {
			tstAsset, ok := asset.(TstReleaseAsset)
			if !ok {
				t.Fatalf(
					"Failed to cast the release asset to TstReleaseAsset: %v",
					err)
			}

			tstAssetContent := tstAsset.GetContent()

			if tstAsset.GetName() == filepath.Base(firstFile.Name()) {
				if tstAssetContent != firstFileContent {
					t.Fatalf("The contents of the first uploaded release asset " +
						"don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == filepath.Base(secondFile.Name()) {
				if tstAssetContent != secondFileContent {
					t.Fatalf("The contents of the second uploaded release asset " +
						"don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == filepath.Base(thirdFile.Name()) {
				if tstAssetContent != thirdFileContent {
					t.Fatalf("The contents of the third uploaded release asset " +
						"don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == fourthAssetName {
				if tstAssetContent != fourthAssetContent {
					t.Fatalf("The contents of the fourth (non-uploaded) release " +
						"asset don't match the original asset contents")
				}
			} else if tstAsset.GetName() == fifthAssetName {
				if tstAssetContent != fifthAssetContent {
					t.Fatalf("The contents of the fifth (non-uploaded) release " +
						"asset don't match the original asset contents")
				}
			} else {
				t.Fatalf("Found unidentified release asset: %+v", tstAsset)
			}
		}
	}
}

func TestDeletionOfPreviousReleaseOnTargetCommitMismatch(t *testing.T) {
	binaryContent := "Binary content"
	file, err := setupSampleAssetFile("singleUploadedBinary.txt", binaryContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the single "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(file.Name())
	defer file.Close()

	oldCommit := generateRandomString(16)
	oldId := int64(0)
	commit := generateRandomString(16)
	branch := "master"
	tag := "continuous-master"
	owner := "d1vanov"
	repo := "ciuploadtool"
	repoSlug := owner + "/" + repo
	isPullRequest := false

	releaseSuffix := "master"
	releaseBody := "Continuous release"

	for i := 0; i < 2; i++ {
		if i == 0 {
			setupTravisCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "Travis CI build log: " +
				"https://travis-ci.org/d1vanov/ciuploadtool/builds/" +
				os.Getenv("TRAVIS_BUILD_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: " +
				"https://ci.appveyor.com/project/" + owner + "/" + repo +
				"/build/" + os.Getenv("APPVEYOR_BUILD_VERSION")
		}

		clientFactory := func(
			gitHubToken string,
			owner string,
			repo string) Client {

			info, err := collectBuildEventInfo(releaseSuffix, false)
			if err != nil {
				panic(err)
			}
			tstRelease := newTstRelease(releaseBody, info, false).(*TstRelease)
			tstRelease.targetCommitish = oldCommit
			tstRelease.tagName = tag
			oldId = tstRelease.GetID()
			tstClient := newTstClient(gitHubToken, owner, repo).(*TstClient)
			tstClient.releases = append(tstClient.releases, *tstRelease)
			tstClient.tagNames = append(tstClient.tagNames, tag)
			return tstClient
		}

		client, err := uploadImpl(
			clientFactoryFunc(clientFactory),
			releaseFactoryFunc(newTstRelease),
			[]string{file.Name()},
			releaseSuffix,
			releaseBody,
			false)
		if err != nil {
			t.Fatalf("Failed to upload the single binary: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Fatalf(
				"Detected wrong number of releases within client: want 1, have %d",
				len(tstClient.releases))
		}

		release := tstClient.releases[0]
		if release.GetTargetCommitish() != commit {
			t.Fatalf("Unexpected target commitish for release")
		}

		if release.GetID() == oldId {
			t.Fatalf("Unexpected id of the resource, equal to the old id while " +
				"expected the id to be new for new release")
		}
	}
}

func TestNewReleaseBuildCreation(t *testing.T) {
	binaryContent := "Binary content"
	file, err := setupSampleAssetFile("singleUploadedBinary.txt", binaryContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the single "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(file.Name())
	defer file.Close()

	// Make the uploader's life harder: add two continuous releases which should
	// not be messed up by adding the new release
	continuousMasterCommit := generateRandomString(16)
	continuousDevCommit := generateRandomString(16)
	continuousMasterTag := "continuous-master"
	continuousDevTag := "continuous-development"
	continuousMasterContent := "continuous-master-content"
	continuousDevContent := "continuous-dev-content"

	commit := generateRandomString(16)
	branch := "master"
	tag := "v1.0.0"
	owner := "d1vanov"
	repo := "ciuploadtool"
	repoSlug := owner + "/" + repo
	isPullRequest := false

	releaseSuffix := tag
	releaseBody := ""

	for i := 0; i < 2; i++ {
		if i == 0 {
			setupTravisCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "Travis CI build log: " +
				"https://travis-ci.org/d1vanov/ciuploadtool/builds/" +
				os.Getenv("TRAVIS_BUILD_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: " +
				"https://ci.appveyor.com/project/" + owner + "/" + repo +
				"/build/" + os.Getenv("APPVEYOR_BUILD_VERSION")
		}

		clientFactory := func(
			gitHubToken string,
			owner string,
			repo string) Client {

			info, err := collectBuildEventInfo(releaseSuffix, false)
			if err != nil {
				panic(err)
			}

			continuousMasterAsset := TstReleaseAsset{
				id:      lastFreeReleaseAssetId,
				name:    filepath.Base(file.Name()),
				content: continuousMasterContent}
			lastFreeReleaseAssetId++

			continuousMasterRelease := newTstRelease(
				releaseBody,
				info,
				false).(*TstRelease)

			continuousMasterRelease.targetCommitish = continuousMasterCommit
			continuousMasterRelease.tagName = continuousMasterTag

			continuousMasterRelease.name = "Continuous release (" +
				continuousMasterTag + ")"

			continuousMasterRelease.isPrerelease = true
			continuousMasterRelease.assets = append(
				continuousMasterRelease.assets,
				continuousMasterAsset)

			continuousDevAsset := TstReleaseAsset{
				id:      lastFreeReleaseAssetId,
				name:    filepath.Base(file.Name()),
				content: continuousDevContent}
			lastFreeReleaseAssetId++

			continuousDevRelease := newTstRelease(
				releaseBody,
				info,
				false).(*TstRelease)

			continuousDevRelease.targetCommitish = continuousDevCommit
			continuousDevRelease.tagName = continuousDevTag

			continuousDevRelease.name = "Continuous release (" +
				continuousDevTag + ")"

			continuousDevRelease.isPrerelease = true
			continuousDevRelease.assets = append(
				continuousDevRelease.assets,
				continuousDevAsset)

			tstClient := newTstClient(gitHubToken, owner, repo).(*TstClient)

			tstClient.releases = append(
				tstClient.releases,
				*continuousMasterRelease)

			tstClient.releases = append(
				tstClient.releases,
				*continuousDevRelease)

			tstClient.tagNames = append(tstClient.tagNames, continuousMasterTag)
			tstClient.tagNames = append(tstClient.tagNames, continuousDevTag)
			return tstClient
		}

		client, err := uploadImpl(
			clientFactoryFunc(clientFactory),
			releaseFactoryFunc(newTstRelease),
			[]string{file.Name()},
			releaseSuffix,
			releaseBody,
			false)
		if err != nil {
			t.Fatalf("Failed to upload the single binary: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 3 {
			t.Fatalf(
				"Detected wrong number of releases within client: want 3, have %d",
				len(tstClient.releases))
		}

		foundContinuousMasterRelease := false
		foundContinuousDevRelease := false

		for _, tstRelease := range tstClient.releases {
			var release Release = &tstRelease
			assets := release.GetAssets()
			if len(assets) != 1 {
				t.Fatalf(
					"Wrong number of assets within the release: want %d, have %d",
					1,
					len(assets))
			}

			asset := assets[0].(TstReleaseAsset)

			if release.GetTagName() == continuousMasterTag {
				foundContinuousMasterRelease = true
				if asset.content != continuousMasterContent {
					t.Fatalf("Detected wrong content for continuous master " +
						"release after creating non-continuous release")
				}
			} else if release.GetTagName() == continuousDevTag {
				foundContinuousDevRelease = true
				if asset.GetContent() != continuousDevContent {
					t.Fatalf("Detected wrong content for continuous dev release " +
						"after creating non-continuous release")
				}
			} else if release.GetTagName() == tag {
				if release.GetPrerelease() {
					t.Fatalf("The non-continuous tagged release is marked as " +
						"prerelease which is not intended")
				} else if asset.GetContent() != binaryContent {
					t.Fatalf("Detected wrong content for official release: want "+
						"%q, have %q", binaryContent, asset.GetContent())
				} else if release.GetTargetCommitish() != commit {
					t.Fatalf(
						"Detected wrong target commit to which the official "+
							"release corresponds: want %q, have %q",
						commit,
						release.GetTargetCommitish())
				}
			}
		}

		if !foundContinuousMasterRelease {
			t.Fatalf("Haven't found the continous master release after creating " +
				"the official release")
		}

		if !foundContinuousDevRelease {
			t.Fatalf("Haven't found the continous development release after " +
				"creating the official release")
		}
	}
}

func TestReleaseAfterBothTravisAndAppVeyorBuildJobs(t *testing.T) {
	travisBinaryContent := "Travis binary content"
	travisFile, err := setupSampleAssetFile(
		"travisUploadedBinary.txt",
		travisBinaryContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the binary "+
			"uploaded from Travis CI: %v", err)
	}

	defer os.Remove(travisFile.Name())
	defer travisFile.Close()

	appVeyorBinaryContent := "AppVeyor binary content"
	appVeyorFile, err := setupSampleAssetFile(
		"appVeyorUploadedBinary.txt",
		appVeyorBinaryContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the binary "+
			"uploaded from AppVeyor CI: %v", err)
	}

	defer os.Remove(appVeyorFile.Name())
	defer appVeyorFile.Close()

	commit := generateRandomString(16)
	branch := "master"
	tag := "continuous-master"
	owner := "d1vanov"
	repo := "ciuploadtool"
	repoSlug := owner + "/" + repo
	isPullRequest := false

	releaseSuffix := "master"
	releaseBody := ""

	client := TstClient{}

	for i := 0; i < 2; i++ {
		if i == 0 {
			setupTravisCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "Travis CI build log: " +
				"https://travis-ci.org/d1vanov/ciuploadtool/builds/" +
				os.Getenv("TRAVIS_BUILD_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: " +
				"https://ci.appveyor.com/project/" + owner + "/" + repo +
				"/build/" + os.Getenv("APPVEYOR_BUILD_VERSION")
		}

		clientFactory := func(
			gitHubToken string,
			owner string,
			repo string) Client {

			_, err := collectBuildEventInfo(releaseSuffix, false)
			if err != nil {
				panic(err)
			}
			client.token = gitHubToken
			client.owner = owner
			client.repo = repo
			return &client
		}

		var file *os.File
		if i == 0 {
			file = travisFile
		} else {
			file = appVeyorFile
		}

		_, err := uploadImpl(
			clientFactoryFunc(clientFactory),
			releaseFactoryFunc(newTstRelease),
			[]string{file.Name()},
			releaseSuffix,
			releaseBody,
			false)
		if err != nil {
			t.Fatalf("Failed to upload one of binaries: %v", err)
		}
	}

	if len(client.releases) != 1 {
		t.Fatalf(
			"Wrong number of releases within the client: want %d, have %d",
			1,
			len(client.releases))
	}

	release := client.releases[0]
	assets := release.GetAssets()
	if len(assets) != 2 {
		t.Fatalf(
			"Wrong number of assets within the release: want %d, have %d",
			2,
			len(assets))
	}

	foundTravisResourceAsset := false
	foundAppVeyorResourceAsset := false
	for _, asset := range assets {
		tstReleaseAsset := asset.(TstReleaseAsset)
		if tstReleaseAsset.content == travisBinaryContent {
			foundTravisResourceAsset = true
		} else if tstReleaseAsset.content == appVeyorBinaryContent {
			foundAppVeyorResourceAsset = true
		}
	}

	if !foundTravisResourceAsset {
		t.Fatalf("Failed to find the release asset corresponding to the binary " +
			"uploaded from Travis CI build")
	}

	if !foundAppVeyorResourceAsset {
		t.Fatalf("Failed to find the release asset corresponding to the binary " +
			"uploaded from AppVeyor CI build")
	}

	foundTravisCiBuildLogLine := false
	foundAppVeyorCiBuildLogLine := false

	scanner := bufio.NewScanner(strings.NewReader(release.GetBody()))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Travis CI build log: "+
			"https://travis-ci.org/"+owner+"/"+repo+"/builds/") {

			if foundTravisCiBuildLogLine {
				t.Fatalf("Found Travis CI build log more than once within " +
					"the release body")
			}
			foundTravisCiBuildLogLine = true
		} else if strings.HasPrefix(line, "AppVeyor CI build log: "+
			"https://ci.appveyor.com/project/"+owner+"/"+repo+"/build") {

			if foundAppVeyorCiBuildLogLine {
				t.Fatalf("Found AppVeyor CI build log more than once within " +
					"the release body")
			}
			foundAppVeyorCiBuildLogLine = true
		}
	}

	if !foundTravisCiBuildLogLine {
		t.Fatalf("Haven't found the Travis CI build log within the release body")
	}

	if !foundAppVeyorCiBuildLogLine {
		t.Fatalf("Haven't found the AppVeyor CI build log with the release body")
	}
}

func TestNewNonContinuousReleaseWithSingleUploadedBinaryWithoutSpecifiedSuffix(
	t *testing.T) {

	binaryContent := "Binary content"
	file, err := setupSampleAssetFile("singleUploadedBinary.txt", binaryContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the single "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(file.Name())
	defer file.Close()

	commit := generateRandomString(16)
	branch := "master"
	tag := "v1.0.0"
	owner := "d1vanov"
	repo := "ciuploadtool"
	repoSlug := owner + "/" + repo
	isPullRequest := false

	releaseSuffix := ""
	releaseBody := ""

	for i := 0; i < 2; i++ {
		if i == 0 {
			setupTravisCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "Travis CI build log: " +
				"https://travis-ci.org/d1vanov/ciuploadtool/builds/" +
				os.Getenv("TRAVIS_BUILD_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: " +
				"https://ci.appveyor.com/project/" + owner + "/" + repo +
				"/build/" + os.Getenv("APPVEYOR_BUILD_VERSION")
		}

		client, err := uploadImpl(
			clientFactoryFunc(newTstClient),
			releaseFactoryFunc(newTstRelease),
			[]string{file.Name()},
			releaseSuffix,
			releaseBody,
			false)
		if err != nil {
			t.Fatalf("Failed to upload the single binary: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Fatalf(
				"Wrong number of releases within the returned client, "+
					"want %d, have %d",
				1,
				len(tstClient.releases))
		}

		release := tstClient.releases[0]
		if release.GetTagName() != tag {
			t.Fatalf(
				"Wrong tag name within the release: want %q, have %q",
				tag,
				release.GetTagName())
		}

		if release.GetPrerelease() {
			t.Fatalf("The created release is prerelease while it was expected " +
				"to be non-prerelease")
		}
	}
}

func TestNewContinuousReleaseWithSingleUploadedBinaryWithoutSpecifiedSuffixWithoutTag(
	t *testing.T) {

	binaryContent := "Binary content"
	file, err := setupSampleAssetFile("singleUploadedBinary.txt", binaryContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the single "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(file.Name())
	defer file.Close()

	commit := generateRandomString(16)
	branch := "master"
	tag := ""
	owner := "d1vanov"
	repo := "ciuploadtool"
	repoSlug := owner + "/" + repo
	isPullRequest := false

	releaseSuffix := ""
	releaseBody := ""

	for i := 0; i < 2; i++ {
		if i == 0 {
			setupTravisCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "Travis CI build log: " +
				"https://travis-ci.org/d1vanov/ciuploadtool/builds/" +
				os.Getenv("TRAVIS_BUILD_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: " +
				"https://ci.appveyor.com/project/" + owner + "/" + repo +
				"/build/" + os.Getenv("APPVEYOR_BUILD_VERSION")
		}

		client, err := uploadImpl(
			clientFactoryFunc(newTstClient),
			releaseFactoryFunc(newTstRelease),
			[]string{file.Name()},
			releaseSuffix,
			releaseBody,
			false)
		if err != nil {
			t.Fatalf("Failed to upload the single binary: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Fatalf("Wrong number of releases within the returned client, "+
				"want %d, have %d", 1, len(tstClient.releases))
		}

		release := tstClient.releases[0]
		if release.GetTagName() != "continuous" {
			t.Fatalf("Wrong tag name within the release: want \"continuous\", "+
				"have %q", release.GetTagName())
		}

		if !release.GetPrerelease() {
			t.Fatalf("The created release is not prerelease while it was " +
				"expected to be prerelease")
		}
	}
}

func TestNewContinuousReleaseWithSingleUploadedBinaryWithoutSpecifiedSuffixWithTag(
	t *testing.T) {

	binaryContent := "Binary content"
	file, err := setupSampleAssetFile("singleUploadedBinary.txt", binaryContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the single "+
			"uploaded binary: %v", err)
	}

	defer os.Remove(file.Name())
	defer file.Close()

	commit := generateRandomString(16)
	branch := "master"
	tag := "continuous"
	owner := "d1vanov"
	repo := "ciuploadtool"
	repoSlug := owner + "/" + repo
	isPullRequest := false

	releaseSuffix := ""
	releaseBody := ""

	for i := 0; i < 2; i++ {
		if i == 0 {
			setupTravisCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "Travis CI build log: " +
				"https://travis-ci.org/d1vanov/ciuploadtool/builds/" +
				os.Getenv("TRAVIS_BUILD_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: " +
				"https://ci.appveyor.com/project/" + owner + "/" + repo +
				"/build/" + os.Getenv("APPVEYOR_BUILD_VERSION")
		}

		client, err := uploadImpl(
			clientFactoryFunc(newTstClient),
			releaseFactoryFunc(newTstRelease),
			[]string{file.Name()},
			releaseSuffix,
			releaseBody,
			false)
		if err != nil {
			t.Fatalf("Failed to upload the single binary: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Fatalf("Wrong number of releases within the returned client, "+
				"want %d, have %d", 1, len(tstClient.releases))
		}

		release := tstClient.releases[0]
		if release.GetTagName() != tag {
			t.Fatalf(
				"Wrong tag name within the release: want %q, have %q",
				tag,
				release.GetTagName())
		}

		if !release.GetPrerelease() {
			t.Fatalf("The created release is not prerelease while it was " +
				"expected to be prerelease")
		}
	}
}

func setupSampleAssetFile(filename, content string) (*os.File, error) {
	file, err := ioutil.TempFile("", "singleUploadedBinary.txt")
	if err != nil {
		return nil, fmt.Errorf("Failed to create the temporary file "+
			"representing the single uploaded binary: %v", err)
	}

	_, err = file.WriteString(content)
	if err != nil {
		return nil, fmt.Errorf("Failed to write the sample content to "+
			"the temporary file: %v", err)
	}

	return file, nil
}

func setupTravisCiEnvVars(
	commit string,
	branch string,
	tag string,
	repoSlug string,
	isPullRequest bool) {

	os.Unsetenv("APPVEYOR")
	os.Setenv("TRAVIS", "true")
	os.Setenv("GITHUB_TOKEN", "fake_token")
	os.Setenv("TRAVIS_BRANCH", branch)
	os.Setenv("TRAVIS_TAG", tag)
	os.Setenv("TRAVIS_COMMIT", commit)
	os.Setenv("TRAVIS_REPO_SLUG", repoSlug)
	os.Setenv("TRAVIS_BUILD_ID", generateRandomString(10))
	if isPullRequest {
		os.Setenv("TRAVIS_EVENT_TYPE", "pull_request")
	} else {
		os.Setenv("TRAVIS_EVENT_TYPE", "non_pull_request")
	}
}

func setupAppVeyorCiEnvVars(
	commit string,
	branch string,
	tag string,
	repoSlug string,
	isPullRequest bool) {

	os.Unsetenv("TRAVIS")
	os.Setenv("APPVEYOR", "True")
	os.Setenv("auth_token", "fake_token")
	os.Setenv("APPVEYOR_REPO_BRANCH", branch)
	os.Setenv("APPVEYOR_REPO_TAG_NAME", tag)
	os.Setenv("APPVEYOR_REPO_COMMIT", commit)
	os.Setenv("APPVEYOR_REPO_NAME", repoSlug)
	os.Setenv("APPVEYOR_BUILD_VERSION", "0.1.0-31")
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
