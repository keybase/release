// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"

	releaseVersion "github.com/keybase/release/version"
)

// EncodeJSON returns JSON (as bytes) for an update
func EncodeJSON(version string, name string, descriptionPath string, src string, URI fmt.Stringer, signaturePath string, dokanCodeX64 string, dokanCodeX86 string) ([]byte, error) {
	update := Update{
		Version: version,
		Name:    name,
		DokanCodeX64: dokanCodeX64,
		DokanCodeX86: dokanCodeX86, 
	}

	// Get published at from version string
	_, _, date, _, err := releaseVersion.Parse(version)
	if err == nil {
		t := ToTime(date)
		update.PublishedAt = &t
	}

	if src != "" && URI != nil {
		fileName := path.Base(src)

		// Or if we can't parse use the src file modification time
		if update.PublishedAt == nil {
			var srcInfo os.FileInfo
			srcInfo, err = os.Stat(src)
			if err != nil {
				return nil, err
			}
			t := ToTime(srcInfo.ModTime())
			update.PublishedAt = &t
		}

		urlString := fmt.Sprintf("%s/%s", URI.String(), url.QueryEscape(fileName))
		asset := Asset{
			Name: fileName,
			URL:  urlString,
		}

		digest, err := digest(src)
		if err != nil {
			return nil, fmt.Errorf("Error creating digest: %s", err)
		}
		asset.Digest = digest

		if signaturePath != "" {
			sig, err := readFile(signaturePath)
			if err != nil {
				return nil, err
			}
			asset.Signature = sig
		}

		if descriptionPath != "" {
			desc, err := readFile(descriptionPath)
			if err != nil {
				return nil, err
			}
			update.Description = desc
		}

		update.Asset = &asset
	}

	return json.MarshalIndent(update, "", "  ")
}

// DecodeJSON returns an update object from JSON (bytes)
func DecodeJSON(r io.Reader) (*Update, error) {
	var obj Update
	if err := json.NewDecoder(r).Decode(&obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

func readFile(path string) (string, error) {
	sigFile, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = sigFile.Close() }()
	data, err := ioutil.ReadAll(sigFile)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func digest(p string) (digest string, err error) {
	hasher := sha256.New()
	f, err := os.Open(p)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	if _, ioerr := io.Copy(hasher, f); ioerr != nil {
		err = ioerr
		return
	}
	digest = hex.EncodeToString(hasher.Sum(nil))
	return
}
