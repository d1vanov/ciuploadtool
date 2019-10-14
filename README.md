ciuploadtool
===========

**Command line tool for uploading of binaries from Travis CI and AppVeyor CI builds to GitHub releases**

Travis CI (Linux, OS X): [![Build Status](https://travis-ci.org/d1vanov/ciuploadtool.svg?branch=master)](https://travis-ci.org/d1vanov/ciuploadtool)

AppVeyor CI (Windows): [![Build status](https://ci.appveyor.com/api/projects/status/rsid6nlpmj2fq5ux/branch/master?svg=true)](https://ci.appveyor.com/project/d1vanov/ciuploadtool/branch/master)

## What's this

If you use Travis CI and/or AppVeyor CI for your project, you know that it is possible to deploy binaries
built by these services to GitHub releases. However, it is not as simple as one would desire. `ciuploadtool` strives
to make it as simple as possible.

The logic behind `ciuploadtool` was heavily inspired by [this tool](https://github.com/probonopd/uploadtool) working with Travis CI only, not AppVeyor CI. I've been
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
      - |
        if [ "${TRAVIS_OS_NAME}" = "linux" ]; then
          wget https://github.com/d1vanov/ciuploadtool/releases/download/continuous-master/ciuploadtool_linux.zip &&
          unzip ciuploadtool_linux.zip
        else
          wget https://github.com/d1vanov/ciuploadtool/releases/download/continuous-master/ciuploadtool_mac.zip &&
          unzip ciuploadtool_mac.zip
        fi
      - chmod 755 ciuploadtool
      - ./ciuploadtool out/*

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
      matrix:
        - prepare_mode: YES
        # other matrix branches are your builds: simply append prepare_mode: NO to them
        - prepare_mode: NO
          platform: x86
        - prepare_mode: NO
          platform: x64

    install:
      - md c:\ciuploadtool
      - cd c:\ciuploadtool
      - curl -fsSL https://github.com/d1vanov/ciuploadtool/releases/download/continuous-master/ciuploadtool_windows_x86.zip -o ciuploadtool_windows_x86.zip
      - 7z x ciuploadtool_windows_x86.zip
      - <the rest of your install section goes here>

    build_script:
      - if %prepare_mode%==YES c:\ciuploadtool\ciuploadtool.exe -preponly
      - ps: if ($env:prepare_mode -eq "YES") { throw "Failing in order to stop the current build matrix job early" }
      - <the rest of your build script goes here>

    on_finish:
      - dir /s out\ # Assuming you have some artifacts in out that you would like to upload
      - c:\ciuploadtool\ciuploadtool.exe out\*

    branches:
      except:
        - # Do not build tags that we create when we upload to GitHub Releases
        - /^(?i:continuous)$/

    matrix:
      allow_failures:
        - prepare_mode: YES
```

Note that `ciuploadtool` replaces the normal deployment step for AppVeyor CI, so if you have deployment set up for GitHub,
you should just remove it to prevent conflicts and/or races between AppVeyor's built-in deployment processing and the tool's job.

Also note the tricky setup for AppVeyor's build matrices. The explanation for this trickery is as follows: when the build is
triggered by new commit(s), the tool deletes any previous continuous release which target commit mismatches the latest pushed
commit and creates a new release instead. The creation of a release involves the creation of a tag and apparently AppVeyor CI,
unlike Travis CI, reacts on the creation of a new tag *during* the CI build. Without special tricks that would lead to
duplicate builds performed by AppVeyor CI: first it would build things for the new commit, then as a part of binaries uploading
process `ciuploadtool` would create a new tag for the new continuous release and AppVeyor would schedule another build.
This build would be pretty useless because it would correspond to the very same version of the source code so running that
build would simply waste AppVeyor CI resources and your time.

So here's what's done to prevent such situation: `ciuploadtool` accepts `-preponly` flag which makes the tool perform
all the necessary GitHub release preparation for binaries uploading but without actual binaries uploading. In this mode
the tool would ensure the continuous release's target commit corresponds to the latest pushed commit and if it's not so,
the tool deletes the existing release and creates a new one. The creation of a new release involves the creation of the
continuous tag for this release as well and that is what triggers AppVeyor CI to schedule another build. So we do the following:

 * Do this release preparation before any actual build
 * Do this release preparation in a separate matrix branch
 * Use special trick to force this separate matrix branch job to fail so that it's guaranteed to finish quickly. For that reason we also have `allow_failures` section in the above example.

One other thing necessary for this whole workaround to work is to enable [rolling builds](https://www.appveyor.com/docs/build-configuration/#rolling-builds)
in your project's settings by toggling the checkbox at `https://ci.appveyor.com/project/yourusername/yourrepository/settings`.
The feature would cancel the current build if a new one was scheduled. So the first branch of your build matrix would
run `ciuploadtool` in preparation mode, that would create a new tag which would cause the scheduling of a new build. That job
would then quickly fail (because we specifically make it so) but as long as this build matrix branch allows for failures,
AppVeyor CI won't panic and e-mail you about the broken build. Instead it would just silently cancel the rest of the build matrix's
jobs (due to rolling builds feature) and will switch to the newly scheduled build. That newly scheduled build won't cause
the recreation of a tag because the existing release's target commit would match the expected one - so first build matrix's
branch would do nothing and silently fail and the rest of build matrix's jobs would actually build your project and upload
the binaries to the GitHub release.

## Advanced usage

The tool accepts several input parameters which can be used to fine-tune its behaviour. For example, you might want to
differentiate continuous builds between stable (master branch) and unstable (development branch) builds. This can be done
by adjusting the name of the tag used for continuous builds: `ciuploadtool` accepts input flag `suffix` which can be set equal
to branch name so builds from master branch would produce continuous release tagged with `continuous-master` tag and
builds from development branch would produce continuous release tagged with `continuous-development` tag. Here's example
configuration for Travis CI:

```yaml
    after_success:
      - ls -lh out/* # Assuming you have some files in out/ that you would like to upload
      - |
        if [ "${TRAVIS_OS_NAME}" = "linux" ]; then
          wget https://github.com/d1vanov/ciuploadtool/releases/download/continuous-master/ciuploadtool_linux.zip &&
          unzip ciuploadtool_linux.zip
        else
          wget https://github.com/d1vanov/ciuploadtool/releases/download/continuous-master/ciuploadtool_mac.zip &&
          unzip ciuploadtool_mac.zip
        fi
      - chmod 755 ciuploadtool
      - ./ciuploadtool -suffix="$TRAVIS_BRANCH" out/*

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
      matrix:
        - prepare_mode: YES
        # other matrix branches are your builds: simply append prepare_mode: NO to them
        - prepare_mode: NO
          platform: x86
        - prepare_mode: NO
          platform: x64

    install:
      - md c:\ciuploadtool
      - cd c:\ciuploadtool
      - curl -fsSL https://github.com/d1vanov/ciuploadtool/releases/download/continuous-master/ciuploadtool_windows_x86.zip -o ciuploadtool_windows_x86.zip
      - 7z x ciuploadtool_windows_x86.zip
      - <the rest of your install section goes here>

    build_script:
      - if %prepare_mode%==YES c:\ciuploadtool\ciuploadtool.exe -preponly -suffix="%APPVEYOR_REPO_BRANCH%"
      - ps: if ($env:prepare_mode -eq "YES") { throw "Failing in order to stop the current build matrix job early" }
      - <the rest of your build script goes here>

    on_finish:
      - dir /s out\ # Assuming you have some artifacts in out that you would like to upload
      - c:\ciuploadtool\ciuploadtool.exe -suffix="%APPVEYOR_REPO_BRANCH%" out\*

    branches:
      only:
        - master
        - development
        - /^v\d+\.\d+(\.\d+)?(-\S*)?$/

    matrix:
      allow_failures:
        - prepare_mode: YES
```

Note also that this scheme uses one subtle particularity of both Travis CI and AppVeyor CI: `TRAVIS_BRANCH` and `APPVEYOR_REPO_BRANCH`
are equal to actual branch names during builds produces by commits but they are equal to tag names when builds are triggered
by pushed tags. I.e. when you do something like `git tag -a v1.0.0 -m "My first release" && git push origin v1.0.0`, that action triggers
build with branch name set to the name of the pushed tag; `ciuploadtool` checks whether the values of these environment variables
match the name of the pushed tag (if any) and if so, it creates a non-continuous release to which the specified binaries are uploaded
in precisely the same way as for continuous builds.

And the last note is about the processing of binaries produced by different branches of the build matrix: you can upload binaries for each
branch of the build matrix thus providing your users with freedom to choose the build for download among several available builds -
either built using different toolsets or built in different configurations etc. `ciuploadtool` associates the releases it creates
with commits so it won't create more than one release for the given commit. All the specified binaries from all builds corresponding
to the release would be uploaded to that single release. That has one particularity: **you better configure the names of the binaries you upload
to NOT be different between different builds corresponding to the same commit**. For example, you would violate this rule if you make the name
of your uploadable binary contain `APPVEYOR_BUILD_ID` which is the id of AppVeyor build job different for each build or `APPVEYOR_BUILD_NUMBER`
which is incremented with each consequent build, regardless of which commits it corresponds to. If you make the names of your binaries different
for each build **and** repeat builds for the same commit multiple times (both Travis CI and AppVeyor allow to trigger builds for the same commit
multiple times), `ciuploadtool` would attach more and more binaries with different names to the same release. However, if you binaries have names
corresponding to commits insteaf of build job ids or numbers, `ciuploadtool` would replace older binaries, if they were attached
to the given release previously, with newer ones.

You can check out [this test project](https://github.com/d1vanov/ciuploadtool-testing) used for testing of `ciuploadtool` and see how things are organized there.
