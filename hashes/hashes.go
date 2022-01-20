package hashes

import (
	"crypto/sha1"
	"encoding/json"
	"os"

	"golang.org/x/mod/sumdb/dirhash"

	"github.com/marmotherder/habitable/common"
)

type hashData struct {
	Files       map[string]string `json:"files"`
	Directories map[string]string `json:"directories"`
}

func loadHashes() (hashData, error) {
	dirHashes := hashData{
		Files:       map[string]string{},
		Directories: map[string]string{},
	}
	hashesFile := common.TempBuildDir() + "/" + "hashes.json"
	if _, fileErr := os.Stat(hashesFile); fileErr == nil {
		common.AppLogger.Debug("opening hashes.json")
		contents, err := os.ReadFile(hashesFile)
		if err != nil {
			common.AppLogger.Error("failed to open existing hashes.json")
			return dirHashes, err
		}
		common.AppLogger.Debug("hashes.json opened successfully")

		common.AppLogger.Debug("parsing hashes.json")
		if err := json.Unmarshal(contents, &dirHashes); err != nil {
			common.AppLogger.Error("failed to read existing hashes.json")
			return dirHashes, err
		}
		common.AppLogger.Debug("hashes.json parsed successfully")
	}

	common.AppLogger.Trace("hashes.json contents:")
	common.AppLogger.Trace(dirHashes)
	return dirHashes, nil
}

func updateHashesFile(hashes hashData) error {
	common.AppLogger.Debug("writing updates to hashes.json")
	updatedHashes, err := json.Marshal(hashes)
	if err != nil {
		common.AppLogger.Error("failed to marshal updated hashes")
		return err
	}

	hashesFile := common.TempBuildDir() + "/" + "hashes.json"
	if err = os.WriteFile(hashesFile, updatedHashes, 0640); err != nil {
		common.AppLogger.Error("failed to update hashes.json")
		return err
	}

	common.AppLogger.Debug("hashes.json updated successfully")
	common.AppLogger.Trace(string(updatedHashes))

	return nil
}

func CheckDirectoryHashes(directories ...string) (hasChanges bool, err error) {
	hashes, loadErr := loadHashes()
	if loadErr != nil {
		err = loadErr
		return
	}

	common.AppLogger.Debug("checking hashes for %s", directories)
	for _, directory := range directories {
		dirhash, hashErr := dirhash.HashDir(directory, "", dirhash.DefaultHash)
		if err != nil {
			common.AppLogger.Error("failed to hash %s", directory)
			err = hashErr
			return
		}
		common.AppLogger.Trace("got the following hash for directory %s: %s", directory, dirhash)

		if existingDirHash, ok := hashes.Directories[directory]; ok {
			if existingDirHash != dirhash {
				common.AppLogger.Trace("directory %s has been changed", directory)
				hasChanges = true
			}
		} else {
			common.AppLogger.Trace("directory %s is newly requested", directory)
			hasChanges = true
		}

		hashes.Directories[directory] = dirhash
	}

	if !hasChanges {
		common.AppLogger.Debug("no changes found in hashes, continuing")
		return
	}

	common.AppLogger.Debug("changes in hashes found, updating hashes.json")
	if err = updateHashesFile(hashes); err != nil {
		return
	}

	return
}

func CheckStringHash(id, input string) (hasChanges bool, err error) {
	hashes, loadErr := loadHashes()
	if err != nil {
		err = loadErr
		return
	}

	h := sha1.New()
	h.Write([]byte(input))
	bs := string(h.Sum(nil))
	common.AppLogger.Trace("got the following hash for id %s: %s", id, bs)

	if existing, ok := hashes.Files[id]; ok {
		if existing != bs {
			common.AppLogger.Trace("string id %s has been changed", id)
			hasChanges = true
		}
	} else {
		common.AppLogger.Trace("string id %s is newly requested", id)
		hasChanges = true
	}

	hashes.Files[id] = bs

	if !hasChanges {
		common.AppLogger.Debug("no changes found in hashes, continuing")
		return
	}

	common.AppLogger.Debug("changes in hashes found, updating hashes.json")
	if err = updateHashesFile(hashes); err != nil {
		return
	}

	return
}
