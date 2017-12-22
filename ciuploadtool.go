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

	flag.Parse()

	if !prepareOnly && flag.NArg() < 1 {
		fmt.Printf("Usage: %s [-suffix=<suffix for continuous release names>] [-relbody=<release body message>] <files to upload>\n", os.Args[0])
		os.Exit(-1)
	}

	err := uploader.Upload(flag.Args(), releaseSuffix, releaseBody)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
