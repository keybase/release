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

	"github.com/keybase/client/go/protocol"
	releaseVersion "github.com/keybase/release/version"
)

func EncodeJSON(version string, name string, src string, URI *url.URL, signature string) ([]byte, error) {
	update := keybase1.Update{
		Version: version,
		Name:    name,
	}

	if src != "" && URI != nil {
		fileName := path.Base(src)

		// Try to get public at from version string
		_, date, _, err := releaseVersion.Parse(fileName)
		if err == nil {
			t := keybase1.ToTime(date)
			update.PublishedAt = &t
		}

		// Or use src file modification time for PublishedAt
		if update.PublishedAt == nil {
			var srcInfo os.FileInfo
			srcInfo, err = os.Stat(src)
			if err != nil {
				return nil, err
			}
			t := keybase1.ToTime(srcInfo.ModTime())
			update.PublishedAt = &t
		}

		urlString := fmt.Sprintf("%s/%s", URI.String(), url.QueryEscape(fileName))
		asset := keybase1.Asset{
			Name: fileName,
			Url:  urlString,
		}

		digest, err := digest(src)
		if err != nil {
			return nil, fmt.Errorf("Error creating digest: %s", err)
		} else {
			asset.Digest = digest
		}

		if signature != "" {
			sig, err := readFile(signature)
			if err != nil {
				return nil, err
			}
			asset.Signature = sig
		}

		update.Asset = &asset
	}

	return json.MarshalIndent(update, "", "  ")
}

func DecodeJSON(b []byte) (*keybase1.Update, error) {
	var obj keybase1.Update
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

func readFile(path string) (string, error) {
	sigFile, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer sigFile.Close()
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
	defer f.Close()
	if _, ioerr := io.Copy(hasher, f); ioerr != nil {
		err = ioerr
		return
	}
	digest = hex.EncodeToString(hasher.Sum(nil))
	return
}
