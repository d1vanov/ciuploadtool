package main

import (
	"flag"
	"fmt"
	"github.com/d1vanov/ciuploadtool/uploader"
	"os"
)

func main() {
	var releaseSuffix string
	flag.StringVar(&releaseSuffix, "suffix", "", "Optional suffix for the names of created continuous releases")

	var releaseBody string
	flag.StringVar(&releaseBody, "relbody", "", "Optional content for the body of created releases")

	var prepareOnly bool
	flag.BoolVar(&prepareOnly, "preponly", false, "Specify this flag and no artifacts for uploading to prepare the release for binaries uploading")

	var uploadToService string
	flag.StringVar(&uploadToService, "upload-to-service", "", "Which service to upload the binaries to; if empty, the same service on which the build is running is used")

	var uploadToRepo string
	flag.StringVar(&uploadToRepo, "upload-to-repo", "", "Which repo to upload the binaries to; if empty, the same repo the commit to which triggered the build is used")

	var uploadToRepoOwner string
	flag.StringVar(&uploadToRepoOwner, "upload-to-repo-owner", "", "The owner of the repo to which the binaries are to be uploaded; if empty, the owner of the repo is deduced from the build environment")

	var uploadAuthToken string
	flag.StringVar(&uploadAuthToken, "upload-auth-token", "", "The auth token to be used for uploading of binaries to the release; if empty, the auth token is deduced from the build environment")

	flag.Parse()

	if !prepareOnly && flag.NArg() < 1 {
		fmt.Printf("Usage: %s [-suffix=<suffix for continuous release names>] "+
			"[-relbody=<release body message>] [-upload-to-service=<service: github or gitlab>] "+
			"[-upload-to-repo=<repo>] [-upload-to-repo-owner=<owner>] [-upload-auth-token=<token>] <files to upload>\n", os.Args[0])
		os.Exit(-1)
	}

	var err error
	if prepareOnly {
		fmt.Println("Prepare only flag is active, won't upload any real binaries, will just prepare the release")
		err = uploader.Upload([]string{}, uploadToService, uploadToRepo, uploadToRepoOwner, uploadAuthToken, releaseSuffix, releaseBody)
	} else {
		err = uploader.Upload(flag.Args(), uploadToService, uploadToRepo, uploadToRepoOwner, uploadAuthToken, releaseSuffix, releaseBody)
	}

	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
