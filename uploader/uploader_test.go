package uploader

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
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
		t.Fatalf("Failed to create the temporary file representing the single uploaded binary: %v", err)
	}

	defer os.Remove(file.Name())
	defer file.Close()

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

		asset := assets[0]
		tstAsset, ok := asset.(TstReleaseAsset)
		if !ok {
			t.Fatalf("Failed to cast the release asset to TstReleaseAsset: %v", err)
		}

		tstAssetContent := tstAsset.GetContent()
		if tstAssetContent != binaryContent {
			t.Errorf("The contents of uploaded release asset don't match the original resource file's contents")
		}
	}
}

func TestNewReleaseWithSeveralUploadedBinaries(t *testing.T) {
	firstFileContent := "First file content"
	firstFile, err := setupSampleAssetFile("firstUploadedBinary.txt", firstFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the first uploaded binary: %v", err)
	}

	defer os.Remove(firstFile.Name())
	defer firstFile.Close()

	secondFileContent := "Second file content"
	secondFile, err := setupSampleAssetFile("secondUploadedBinary.txt", secondFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the second uploaded binary: %v", err)
	}

	defer os.Remove(secondFile.Name())
	defer secondFile.Close()

	thirdFileContent := "Third file content"
	thirdFile, err := setupSampleAssetFile("thirdUploadedBinary.txt", thirdFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the third uploaded binary: %v", err)
	}

	defer os.Remove(thirdFile.Name())
	defer thirdFile.Close()

	commit := generateRandomString(16)
	branch := "master"
	tag := "continuous-master"
	repoSlug := "d1vanov/ciuploadtool"
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
			releaseBody = "Travis CI build log: https://travis-ci.org/d1vanov/ciuploadtool/builds/" + os.Getenv("TRAVIS_JOB_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: https://ci.appveyor.com/api/buildjobs/" + os.Getenv("APPVEYOR_JOB_ID") + "/log"
		}

		client, err := uploadImpl(clientFactoryFunc(newTstClient), releaseFactoryFunc(newTstRelease), filenames,
			releaseSuffix, releaseBody)
		if err != nil {
			t.Errorf("Failed to upload one of binaries: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Errorf("Uploading one of binaries to new release failed: no releases within the returned client")
		}

		release := tstClient.releases[0]
		assets := release.GetAssets()
		if len(assets) != 3 {
			t.Errorf("Uploading one of binaries to new release failed: wrong number of assets within the release")
		}

		for _, asset := range assets {
			tstAsset, ok := asset.(TstReleaseAsset)
			if !ok {
				t.Fatalf("Failed to cast the release asset to TstReleaseAsset: %v", err)
			}

			tstAssetContent := tstAsset.GetContent()

			if tstAsset.GetName() == filepath.Base(firstFile.Name()) {
				if tstAssetContent != firstFileContent {
					t.Errorf("The contents of the first uploaded release asset don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == filepath.Base(secondFile.Name()) {
				if tstAssetContent != secondFileContent {
					t.Errorf("The contents of the second uploaded release asset don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == filepath.Base(thirdFile.Name()) {
				if tstAssetContent != thirdFileContent {
					t.Errorf("The contents of the third uploaded release asset don't match the original resource file's contents")
				}
			} else {
				t.Errorf("Found unidentified release asset: %+v", tstAsset)
			}
		}
	}
}

func TestInitiallyEmptyExistingReleaseWithSingleUploadedBinary(t *testing.T) {
	binaryContent := "Binary content"
	file, err := setupSampleAssetFile("singleUploadedBinary.txt", binaryContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the single uploaded binary: %v", err)
	}

	defer os.Remove(file.Name())
	defer file.Close()

	commit := generateRandomString(16)
	branch := "master"
	tag := "continuous-master"
	repoSlug := "d1vanov/ciuploadtool"
	isPullRequest := false

	releaseSuffix := "master"
	releaseBody := "Continuous release"

	for i := 0; i < 2; i++ {
		if i == 0 {
			setupTravisCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "Travis CI build log: https://travis-ci.org/d1vanov/ciuploadtool/builds/" + os.Getenv("TRAVIS_JOB_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: https://ci.appveyor.com/api/buildjobs/" + os.Getenv("APPVEYOR_JOB_ID") + "/log"
		}

		clientFactory := func(gitHubToken string, owner string, repo string) Client {
			info, err := collectBuildEventInfo(releaseSuffix)
			if err != nil {
				panic(err)
			}
			tstRelease := newTstRelease(releaseBody, info).(*TstRelease)
			tstClient := newTstClient(gitHubToken, owner, repo).(*TstClient)
			tstClient.releases = append(tstClient.releases, *tstRelease)
			return tstClient
		}

		client, err := uploadImpl(clientFactoryFunc(clientFactory), releaseFactoryFunc(newTstRelease), []string{file.Name()},
			releaseSuffix, releaseBody)
		if err != nil {
			t.Errorf("Failed to upload one of binaries: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Errorf("Uploading one of binaries to existing release failed: no releases within the returned client")
		}

		release := tstClient.releases[0]
		assets := release.GetAssets()
		if len(assets) != 1 {
			t.Errorf("Uploading one of binaries to existing release failed: no assets within the release")
		}

		asset := assets[0]
		tstAsset, ok := asset.(TstReleaseAsset)
		if !ok {
			t.Fatalf("Failed to cast the release asset to TstReleaseAsset: %v", err)
		}

		tstAssetContent := tstAsset.GetContent()
		if tstAssetContent != binaryContent {
			t.Errorf("The contents of uploaded release asset don't match the original resource file's contents")
		}
	}
}

func TestInitiallyEmptyExistingReleaseWithSeveralUploadedBinaries(t *testing.T) {
	firstFileContent := "First file content"
	firstFile, err := setupSampleAssetFile("firstUploadedBinary.txt", firstFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the first uploaded binary: %v", err)
	}

	defer os.Remove(firstFile.Name())
	defer firstFile.Close()

	secondFileContent := "Second file content"
	secondFile, err := setupSampleAssetFile("secondUploadedBinary.txt", secondFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the second uploaded binary: %v", err)
	}

	defer os.Remove(secondFile.Name())
	defer secondFile.Close()

	thirdFileContent := "Third file content"
	thirdFile, err := setupSampleAssetFile("thirdUploadedBinary.txt", thirdFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the third uploaded binary: %v", err)
	}

	defer os.Remove(thirdFile.Name())
	defer thirdFile.Close()

	commit := generateRandomString(16)
	branch := "master"
	tag := "continuous-master"
	repoSlug := "d1vanov/ciuploadtool"
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
			releaseBody = "Travis CI build log: https://travis-ci.org/d1vanov/ciuploadtool/builds/" + os.Getenv("TRAVIS_JOB_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: https://ci.appveyor.com/api/buildjobs/" + os.Getenv("APPVEYOR_JOB_ID") + "/log"
		}

		clientFactory := func(gitHubToken string, owner string, repo string) Client {
			info, err := collectBuildEventInfo(releaseSuffix)
			if err != nil {
				panic(err)
			}
			tstRelease := newTstRelease(releaseBody, info).(*TstRelease)
			tstClient := newTstClient(gitHubToken, owner, repo).(*TstClient)
			tstClient.releases = append(tstClient.releases, *tstRelease)
			return tstClient
		}

		client, err := uploadImpl(clientFactoryFunc(clientFactory), releaseFactoryFunc(newTstRelease), filenames,
			releaseSuffix, releaseBody)
		if err != nil {
			t.Errorf("Failed to upload one of binaries: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Errorf("Uploading one of binaries to existing release failed: no releases within the returned client")
		}

		release := tstClient.releases[0]
		assets := release.GetAssets()
		if len(assets) != 3 {
			t.Errorf("Uploading one of binaries to existing release failed: wrong number of assets within the release")
		}

		for _, asset := range assets {
			tstAsset, ok := asset.(TstReleaseAsset)
			if !ok {
				t.Fatalf("Failed to cast the release asset to TstReleaseAsset: %v", err)
			}

			tstAssetContent := tstAsset.GetContent()

			if tstAsset.GetName() == filepath.Base(firstFile.Name()) {
				if tstAssetContent != firstFileContent {
					t.Errorf("The contents of the first uploaded release asset don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == filepath.Base(secondFile.Name()) {
				if tstAssetContent != secondFileContent {
					t.Errorf("The contents of the second uploaded release asset don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == filepath.Base(thirdFile.Name()) {
				if tstAssetContent != thirdFileContent {
					t.Errorf("The contents of the third uploaded release asset don't match the original resource file's contents")
				}
			} else {
				t.Errorf("Found unidentified release asset: %+v", tstAsset)
			}
		}
	}
}

func TestExistingReleaseWithSingleUploadedBinary(t *testing.T) {
	binaryContent := "Binary content"
	file, err := setupSampleAssetFile("singleUploadedBinary.txt", binaryContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the single uploaded binary: %v", err)
	}

	defer os.Remove(file.Name())
	defer file.Close()

	commit := generateRandomString(16)
	branch := "master"
	tag := "continuous-master"
	repoSlug := "d1vanov/ciuploadtool"
	isPullRequest := false

	releaseSuffix := "master"
	releaseBody := "Continuous release"

	for i := 0; i < 2; i++ {
		if i == 0 {
			setupTravisCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "Travis CI build log: https://travis-ci.org/d1vanov/ciuploadtool/builds/" + os.Getenv("TRAVIS_JOB_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: https://ci.appveyor.com/api/buildjobs/" + os.Getenv("APPVEYOR_JOB_ID") + "/log"
		}

		clientFactory := func(gitHubToken string, owner string, repo string) Client {
			info, err := collectBuildEventInfo(releaseSuffix)
			if err != nil {
				panic(err)
			}
			tstRelease := newTstRelease(releaseBody, info).(*TstRelease)
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

		client, err := uploadImpl(clientFactoryFunc(clientFactory), releaseFactoryFunc(newTstRelease), []string{file.Name()},
			releaseSuffix, releaseBody)
		if err != nil {
			t.Errorf("Failed to upload the single binary: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Errorf("Uploading single binary to existing release failed: no releases within the returned client")
		}

		release := tstClient.releases[0]
		assets := release.GetAssets()
		if len(assets) != 1 {
			t.Errorf("Uploading single binary to existing release failed: no assets within the release")
		}

		asset := assets[0]
		tstAsset, ok := asset.(TstReleaseAsset)
		if !ok {
			t.Fatalf("Failed to cast the release asset to TstReleaseAsset: %v", err)
		}

		tstAssetContent := tstAsset.GetContent()
		if tstAssetContent != binaryContent {
			t.Errorf("The contents of uploaded release asset don't match the original resource file's contents")
		}
	}
}

func TestExistingReleaseWithSeveralUploadedBinariesAllBeingReplacements(t *testing.T) {
	firstFileContent := "First file content"
	firstFile, err := setupSampleAssetFile("firstUploadedBinary.txt", firstFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the first uploaded binary: %v", err)
	}

	defer os.Remove(firstFile.Name())
	defer firstFile.Close()

	secondFileContent := "Second file content"
	secondFile, err := setupSampleAssetFile("secondUploadedBinary.txt", secondFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the second uploaded binary: %v", err)
	}

	defer os.Remove(secondFile.Name())
	defer secondFile.Close()

	thirdFileContent := "Third file content"
	thirdFile, err := setupSampleAssetFile("thirdUploadedBinary.txt", thirdFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the third uploaded binary: %v", err)
	}

	defer os.Remove(thirdFile.Name())
	defer thirdFile.Close()

	commit := generateRandomString(16)
	branch := "master"
	tag := "continuous-master"
	repoSlug := "d1vanov/ciuploadtool"
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
			releaseBody = "Travis CI build log: https://travis-ci.org/d1vanov/ciuploadtool/builds/" + os.Getenv("TRAVIS_JOB_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: https://ci.appveyor.com/api/buildjobs/" + os.Getenv("APPVEYOR_JOB_ID") + "/log"
		}

		clientFactory := func(gitHubToken string, owner string, repo string) Client {
			info, err := collectBuildEventInfo(releaseSuffix)
			if err != nil {
				panic(err)
			}
			tstRelease := newTstRelease(releaseBody, info).(*TstRelease)
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

		client, err := uploadImpl(clientFactoryFunc(clientFactory), releaseFactoryFunc(newTstRelease), filenames,
			releaseSuffix, releaseBody)
		if err != nil {
			t.Errorf("Failed to upload one of binaries: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Errorf("Uploading one of binaries to existing release failed: no releases within the returned client")
		}

		release := tstClient.releases[0]
		assets := release.GetAssets()
		if len(assets) != 3 {
			t.Errorf("Uploading one of binaries to existing release failed: no assets within the release")
		}

		for _, asset := range assets {
			tstAsset, ok := asset.(TstReleaseAsset)
			if !ok {
				t.Fatalf("Failed to cast the release asset to TstReleaseAsset: %v", err)
			}

			tstAssetContent := tstAsset.GetContent()

			if tstAsset.GetName() == filepath.Base(firstFile.Name()) {
				if tstAssetContent != firstFileContent {
					t.Errorf("The contents of the first uploaded release asset don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == filepath.Base(secondFile.Name()) {
				if tstAssetContent != secondFileContent {
					t.Errorf("The contents of the second uploaded release asset don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == filepath.Base(thirdFile.Name()) {
				if tstAssetContent != thirdFileContent {
					t.Errorf("The contents of the third uploaded release asset don't match the original resource file's contents")
				}
			} else {
				t.Errorf("Found unidentified release asset: %+v", tstAsset)
			}
		}
	}
}

func TestExistingReleaseWithSeveralUploadedBinariesNotAllBeingReplacements(t *testing.T) {
	firstFileContent := "First file content"
	firstFile, err := setupSampleAssetFile("firstUploadedBinary.txt", firstFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the first uploaded binary: %v", err)
	}

	defer os.Remove(firstFile.Name())
	defer firstFile.Close()

	secondFileContent := "Second file content"
	secondFile, err := setupSampleAssetFile("secondUploadedBinary.txt", secondFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the second uploaded binary: %v", err)
	}

	defer os.Remove(secondFile.Name())
	defer secondFile.Close()

	thirdFileContent := "Third file content"
	thirdFile, err := setupSampleAssetFile("thirdUploadedBinary.txt", thirdFileContent)
	if err != nil {
		t.Fatalf("Failed to create the temporary file representing the third uploaded binary: %v", err)
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
	repoSlug := "d1vanov/ciuploadtool"
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
			releaseBody = "Travis CI build log: https://travis-ci.org/d1vanov/ciuploadtool/builds/" + os.Getenv("TRAVIS_JOB_ID") + "/"
		} else {
			setupAppVeyorCiEnvVars(commit, branch, tag, repoSlug, isPullRequest)
			releaseBody = "AppVeyor CI build log: https://ci.appveyor.com/api/buildjobs/" + os.Getenv("APPVEYOR_JOB_ID") + "/log"
		}

		clientFactory := func(gitHubToken string, owner string, repo string) Client {
			info, err := collectBuildEventInfo(releaseSuffix)
			if err != nil {
				panic(err)
			}
			tstRelease := newTstRelease(releaseBody, info).(*TstRelease)
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

		client, err := uploadImpl(clientFactoryFunc(clientFactory), releaseFactoryFunc(newTstRelease), filenames,
			releaseSuffix, releaseBody)
		if err != nil {
			t.Errorf("Failed to upload one of binaries: %v", err)
		}

		tstClient, ok := client.(*TstClient)
		if !ok {
			t.Fatalf("Failed to cast the client to TstClient: %v", err)
		}

		if len(tstClient.releases) != 1 {
			t.Errorf("Uploading one of binaries to existing release failed: no releases within the returned client")
		}

		release := tstClient.releases[0]
		assets := release.GetAssets()
		if len(assets) != 5 {
			t.Errorf("Uploading one of binaries to existing release failed: wrong number of assets within the release")
		}

		for _, asset := range assets {
			tstAsset, ok := asset.(TstReleaseAsset)
			if !ok {
				t.Fatalf("Failed to cast the release asset to TstReleaseAsset: %v", err)
			}

			tstAssetContent := tstAsset.GetContent()

			if tstAsset.GetName() == filepath.Base(firstFile.Name()) {
				if tstAssetContent != firstFileContent {
					t.Errorf("The contents of the first uploaded release asset don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == filepath.Base(secondFile.Name()) {
				if tstAssetContent != secondFileContent {
					t.Errorf("The contents of the second uploaded release asset don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == filepath.Base(thirdFile.Name()) {
				if tstAssetContent != thirdFileContent {
					t.Errorf("The contents of the third uploaded release asset don't match the original resource file's contents")
				}
			} else if tstAsset.GetName() == fourthAssetName {
				if tstAssetContent != fourthAssetContent {
					t.Errorf("The contents of the fourth (non-uploaded) release asset don't match the original asset contents")
				}
			} else if tstAsset.GetName() == fifthAssetName {
				if tstAssetContent != fifthAssetContent {
					t.Errorf("The contents of the fifth (non-uploaded) release asset don't match the original asset contents")
				}
			} else {
				t.Errorf("Found unidentified release asset: %+v", tstAsset)
			}
		}
	}
}

func setupSampleAssetFile(filename, content string) (*os.File, error) {
	file, err := ioutil.TempFile("", "singleUploadedBinary.txt")
	if err != nil {
		return nil, fmt.Errorf("Failed to create the temporary file representing the single uploaded binary: %v", err)
	}

	_, err = file.WriteString(content)
	if err != nil {
		return nil, fmt.Errorf("Failed to write the sample content to the temporary file: %v", err)
	}

	return file, nil
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
