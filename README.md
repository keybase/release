## Release

[![Build Status](https://app.travis-ci.com/keybase/release.svg?branch=master)](https://app.travis-ci.com/github/keybase/release)
[![GoDoc](https://godoc.org/github.com/keybase/release?status.svg)](https://godoc.org/github.com/keybase/release)

This is a command line tool for build and release scripts for generating updates, interacting with Github and S3.

### Example Usage

Generating update.json

```
release update-json --version=1.2.3 --src=/tmp/Keybase.zip --uri=https://s3.amazonaws.com/prerelease.keybase.io/darwin-updates --signature=/tmp/keybase.sig
```
