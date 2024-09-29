package v2

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/barasher/go-exiftool"
)

type Metadata struct {
	Fs   FsMetadata
	Exif ExifMetadata
	Hash HashMetadata
}

func ExtractMetadata(
	et *exiftool.Exiftool,
	base string,
	path string,
) (Metadata, error) {
	fs := ExtractFsMetadata(path) // extracted from the relative path

	fullPath := filepath.Join(base, path)

	exif, err := ExtractExifMetadata(et, fullPath) // extracted from the full path
	if err != nil {
		return Metadata{}, err
	}

	hash, err := ExtractHashMetadata(fullPath) // extracted from the full path
	if err != nil {
		return Metadata{}, err
	}

	return Metadata{
		Fs:   fs,
		Exif: exif,
		Hash: hash,
	}, nil
}

type FsMetadata struct {
	Path string
	Base string
	Ext  string
	Name string
	Dir  string
}

func ExtractFsMetadata(
	file string,
) FsMetadata {
	return FsMetadata{
		Path: file,
		Base: filepath.Base(file),
		Ext:  filepath.Ext(file),
		Name: filepath.Base(file[:len(file)-len(filepath.Ext(file))]),
		Dir:  filepath.Dir(file),
	}
}

type ExifMetadata map[string]interface{}

func ExtractExifMetadata(
	et *exiftool.Exiftool,
	file string,
) (ExifMetadata, error) {
	fms := et.ExtractMetadata(file)
	if len(fms) != 1 {
		return nil, errors.New("cannot extract exif metadata")
	}

	if fms[0].Err != nil {
		return nil, fms[0].Err
	}

	return fms[0].Fields, nil
}

type HashMetadata struct {
	Md5  string
	Sha1 string

	Md5Short  string
	Sha1Short string
}

func ExtractHashMetadata(
	file string,
) (HashMetadata, error) {
	f, err := os.Open(file)
	if err != nil {
		return HashMetadata{}, err
	}
	defer func(f *os.File) { _ = f.Close() }(f)

	hashMd5 := md5.New()
	hashSha1 := sha1.New()
	if _, err := io.Copy(io.MultiWriter(hashMd5, hashSha1), f); err != nil {
		return HashMetadata{}, err
	}

	md5Sum := hashMd5.Sum(nil)
	sha1Sum := hashSha1.Sum(nil)

	md5Str := hex.EncodeToString(md5Sum)
	sha1Str := hex.EncodeToString(sha1Sum)

	return HashMetadata{
		Md5:  md5Str,
		Sha1: sha1Str,

		Md5Short:  md5Str[:8],
		Sha1Short: sha1Str[:8],
	}, nil
}
