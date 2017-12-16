ciuploadtool
===========

**Command line tool for uploading of binaries from Travis CI and AppVeyor CI builds**

## What's this

If you use Travis CI and/or AppVeyor CI for your project, you know that it is possible to deploy the binaries
built by these services to GitHub releases. However, it is not as simple as one would desire. `ciuploadtool` strives
to make it as simple as possible.

The logics behind `ciuploadtool` was heavily inspired by [this tool]() working with Travis CI only, not AppVeyor CI. I've been
using it myself, it's quite easy to use and robust. If you use only Travis CI but not AppVeyor CI, you can just use that solution
and forget about the rest. However, if your project uses AppVeyor CI in addition to Travis CI or exclusively, you might
want to try this tool out.

## WARNING

**At this time the tool is in alpha stage and is not yet ready for production usage, so please DON'T USE IT ON YOUR PRODUCTION BUILDS.
It can screw up your builds and GitHub releases and it can kill your kittens. You were warned.**

## Usage

This tool is designed to be called from Travis CI and/or AppVeyor CI after a successful build. By default, this tool will _delete_
any pre-existing release tagged with `continuous`, tag the current state with the name `continuous`, create a new release with that name,
and upload the specified binaries there.

- On https://github.com/settings/tokens, click on "Generate new token" and generate a token with at least the `public_repo`, `repo:status`, and `repo_deployment` scopes
- On Travis CI, go to the settings of your project at `https://travis-ci.org/yourusername/yourrepository/settings`
- Under "Environment Variables", add key `GITHUB_TOKEN` and the token you generated above as the value. **Make sure that "Display value in build log" is set to "OFF"!**
- In the `.travis.yml` of your GitHub repository, add something like this (assuming the build artifacts to be uploaded are in out/):

    ```yaml
    after_success:
    - ls -lh out/* # Assuming you have some files in out/ that you would like to upload
    - wget -c https://github.com/d1vanov/ciuploadtool/raw/master/ciuploadtool.go
    - go run ciuploadtool.go out/*

    branches:
        except:
            - # Do not build tags that we create when we upload to GitHub Releases
            - /^(?i:continuous)$/
    ```

- On AppVeyor CI, go to the settings of your project at `https://ci.appveyor.com/project/yourusername/yourrepository/settings`
- In the `appveyor.yml` of your GitHub repository, add something like this (assuming the build artifacts to be uploaded are in out\\):

    ```yaml
    before_deploy:
    - dir /s out\ # Assuming you have some artifacts in out that you would like to upload
    - curl -fsSL https://github.com/d1vanov/ciuploadtool/raw/master/ciuploadtool.go -o ciuploadtool.go
    - go run ciuploadtool.go out\*

    branches:
        except:
            - # Do not build tags that we create when we upload to GitHub Releases
            - /^(?i:continuous)$/
    ```

