package extractor

import (
	"crypto/md5"
	"crypto/sha1"
	"github.com/barasher/go-exiftool"
	"io"
	"os"
)

type RawMetadata struct {
	Exif exiftool.FileMetadata
	Hash RawHashInfo
	Path string
}

type RawHashInfo struct {
	Md5  []byte
	Sha1 []byte
}

func getRawMetadata(
	et *exiftool.Exiftool,
	path string,
	exifNeeded bool,
	md5Needed bool,
	sha1Needed bool,
) (RawMetadata, error) {
	res := RawMetadata{
		Path: path,
	}

	if exifNeeded {
		// extract file metadata
		fms := et.ExtractMetadata(path)
		if len(fms) != 1 {
			panic("should not happen")
		}

		res.Exif = fms[0]
	}

	if md5Needed {
		file, err := os.Open(path)
		if err != nil {
			return RawMetadata{}, err
		}
		defer file.Close()

		hasher := md5.New()
		if _, err := io.Copy(hasher, file); err != nil {
			return RawMetadata{}, err
		}

		res.Hash.Md5 = hasher.Sum(nil)
	}

	if sha1Needed {
		file, err := os.Open(path)
		if err != nil {
			return RawMetadata{}, err
		}
		defer file.Close()

		hasher := sha1.New()
		if _, err := io.Copy(hasher, file); err != nil {
			return RawMetadata{}, err
		}

		res.Hash.Sha1 = hasher.Sum(nil)
	}

	return res, nil
}
