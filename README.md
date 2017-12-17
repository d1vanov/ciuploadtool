ciuploadtool
===========

**Command line tool for uploading of binaries from Travis CI and AppVeyor CI builds**

## What's this

If you use Travis CI and/or AppVeyor CI for your project, you know that it is possible to deploy the binaries
built by these services to GitHub releases. However, it is not as simple as one would desire. `ciuploadtool` strives
to make it as simple as possible.

The logics behind `ciuploadtool` was heavily inspired by [this tool]() working with Travis CI only, not AppVeyor CI. I've been
using it myself, it's quite easy to use and robust. If you use only Travis CI but not AppVeyor CI, you can just use that solution
and forget about this project. However, if your project uses AppVeyor CI in addition to Travis CI or exclusively, you might
want to try this tool out.

## Usage

This tool is designed to be called from Travis CI and/or AppVeyor CI after a successful build. By default, this tool will _delete_
any pre-existing release tagged with `continuous`, tag the current state with the name `continuous`, create a new release with that name,
and upload the specified binaries there. Here are the necessary setup steps:

- On https://github.com/settings/tokens, click on "Generate new token" and generate a token with at least the `public_repo`, `repo:status`, and `repo_deployment` scopes
- On Travis CI, go to the settings of your project at `https://travis-ci.org/yourusername/yourrepository/settings`
- Under "Environment Variables", add key `GITHUB_TOKEN` and the token you generated above as the value. **Make sure that "Display value in build log" is set to "OFF"!**
- In the `.travis.yml` of your GitHub repository, add something like this (assuming the build artifacts to be uploaded are in out/):

    ```yaml
    after_success:
      - ls -lh out/* # Assuming you have some files in out/ that you would like to upload
      - wget -c https://github.com/d1vanov/ciuploadtool/raw/master/ciuploadtool.go
      - go get golang.org/x/oauth2 &&
      - go get github.com/google/go-github/github &&
      - go run ciuploadtool.go out/*

    branches:
      except:
        - # Do not build tags that we create when we upload to GitHub Releases
        - /^(?i:continuous)$/
    ```

- On AppVeyor CI, go to page `https://ci.appveyor.com/tools/encrypt`, paste there the GutHub token created above, press "Encrypt" button
- Copy the output encrypted string
- In the `appveyor.yml` of your GitHub repository, add something like this (assuming the build artifacts to be uploaded are in out\\):

    ```yaml
    environment:
      auth_token:
        secure: <your encrypted token> # your encrypted token from GitHub

    on_finish:
      - dir /s out\ # Assuming you have some artifacts in out that you would like to upload
      - curl -fsSL https://github.com/d1vanov/ciuploadtool/raw/master/ciuploadtool.go -o ciuploadtool.go
      - set GOPATH=%cd%
      - go get golang.org/x/oauth2
      - go get github.com/google/go-github/github
      - go run ciuploadtool.go out\*

    branches:
      except:
        - # Do not build tags that we create when we upload to GitHub Releases
        - /^(?i:continuous)$/
    ```

Note that `ciuploadtool` replaces the normal deployment step for AppVeyor CI, so if you have deployment set up for GitHub,
you should just remove it to prevent the conflicts between AppVeyor's built-in deployment processing and the tool's job.

Another helpful tip for AppVeyor is to enable [rolling builds](https://www.appveyor.com/docs/build-configuration/#rolling-builds)
in your project's settings by toggling the checkbox at `https://ci.appveyor.com/project/yourusername/yourrepository/settings`.
The reason to do this is the fact that `ciuploadtool` *might* create a new tag and that by default triggers new build at AppVeyor.
Without rolling builds enabled, you would do two identical builds instead of one - one for commit and one for tag created by the tool
during the commit build. With rolling builds the build which created the new tag would be auto-cancelled in favour of the new build
for the tag.

## Advanced usage

The tool accepts a couple of input parameters which can be used to fine-tune its behaviour. For example, you might want to
differentiate the continuous builds between stable (master branch) and unstable (development branch) builds. This can be done
by adjusting the name of the tag used for continuous builds: `ciuploadtool` accepts input flag `suffix` which can be set equal
to the branch name so builds from master branch would produce continuous release tagged with `continuous-master` tag and
builds from development branch would produce continuous release tagged with `continuous-development` tag. Here's the example
configuration for Travis CI:

    ```yaml
    after_success:
      - ls -lh out/* # Assuming you have some files in out/ that you would like to upload
      - wget -c https://github.com/d1vanov/ciuploadtool/raw/master/ciuploadtool.go
      - go get golang.org/x/oauth2
      - go get github.com/google/go-github/github
      - go run ciuploadtool.go -suffix="$TRAVIS_BRANCH" out/*

    branches:
      only:
        - master
        - development
        - /^v\d+\.\d+(\.\d+)?(-\S*)?$/
    ```

The regular expression within the last line of `branches` list is for tagged releases in the form `v1.0.0` i.e. character `v`
followed by three digits denoting major, minor and patch versions separated by dots. If you name tags some other way, adjust
the regexp accordingly.

The analog of such configuration for AppVeyor CI:

    ```yaml
    environment:
      auth_token:
        secure: <your encrypted token> # your encrypted token from GitHub

    on_finish:
      - dir /s out\ # Assuming you have some artifacts in out that you would like to upload
      - curl -fsSL https://github.com/d1vanov/ciuploadtool/raw/master/ciuploadtool.go -o ciuploadtool.go
      - set GOPATH=%cd%
      - go get golang.org/x/oauth2
      - go get github.com/google/go-github/github
      - go run ciuploadtool.go -suffix="%APPVEYOR_REPO_BRANCH%" out\*

    branches:
      only:
        - master
        - development
        - /^v\d+\.\d+(\.\d+)?(-\S*)?$/
    ```

Note also that this scheme uses one subtle particularity of both Travis CI and AppVeyor CI: `TRAVIS_BRANCH` and `APPVEYOR_REPO_BRANCH`
are equal to the actual branch names during builds produces by commits but they are equal to the names of tags when they are triggered
by pushed tags. I.e. when you do something like `git tag -a v1.0.0 -m "My first release" && git push origin v1.0.0`, that action triggers
builds with branch names set to the name of the pushed tag; `ciuploadtool` checks whether the values of these environment variables
match the name of the pushed tag (if any) and if so, it creates a non-continuous release to which the specified binaries are uploaded
in precisely the same way as for continuous builds.

And the last note is about the processing of binaries produces by different branches of the build matrix: you can upload binaries for each
branch of the build matrix thus providing your users with freedom to choose the build to download builds among several different ones -
either built using different toolset or built in different configurations etc. `ciuploadtool` associates the releases it creates
with commits so it won't create more than one release for the given commit. All the specified binaries from all builds corresponding
to the release would be uploaded to that single release. That has one particularity: you better configure the names of the binaries you upload
to not be different between different builds corresponding to the same commit. For example, you would violate this rule if you make the name
of your uploadable binary contain `APPVEYOR_BUILD_ID` which is the id of AppVeyor build job different for each build or `APPVEYOR_BUILD_NUMBER`
which is incremented with each consequent build, regardless of which commits it corresponds to. If you make the names of your binaries different
for each build **and** repeat builds for the same commit multiple times (both Travis CI and AppVeyor allow to do that), `ciuploadtool` would
attach more and more binaries with different names to the release. However, if you binaries have names corresponding to commits insteaf of
build job ids or numbers, `ciuploadtool` would replace the older binaries, if they were attached to the given release previously, with newer ones.

You can check out [this test project](https://github.com/d1vanov/ciuploadtool-testing) used for testing of `ciuploadtool` and see how things are organized there.
