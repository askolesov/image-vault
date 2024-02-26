package file

import (
	"github.com/askolesov/image-vault/pkg/util"
	util2 "github.com/askolesov/image-vault/pkg2/util"
	"github.com/barasher/go-exiftool"
	"path"
)

func (i *Info) Copy(et *exiftool.Exiftool, libPath string, dryRun bool, log func(string, ...any)) error {
	err := i.GetExifInfo(et, false) // make sure we have the extractor info
	if err != nil {
		return err
	}

	err = i.GetHashInfo(log) // make sure we have the hash info
	if err != nil {
		return err
	}

	if i.IsSidecar && len(i.SidecarFor) > 0 {
		log("Copying sidecar: " + i.Path)

		for _, sidecarFor := range i.SidecarFor {
			err := sidecarFor.GetExifInfo(et, false) // make sure we have the extractor info
			if err != nil {
				return err
			}

			err = sidecarFor.GetHashInfo(log) // make sure we have the hash info
			if err != nil {
				return err
			}

			inLibPath := sidecarFor.GetInLibPath()
			inLibPath = util2.ChangeExtension(inLibPath, i.Extension) // change extension to match sidecar
			targetPath := path.Join(libPath, inLibPath)

			err = util.SmartCopy(i.Path, targetPath, dryRun, log)
			if err != nil {
				return err
			}
		}

		log("Done copying sidecar: " + i.Path)

		return nil
	}

	inLibPath := i.GetInLibPath()
	targetPath := path.Join(libPath, inLibPath)
	err = util.SmartCopy(i.Path, targetPath, dryRun, log)
	if err != nil {
		return err
	}

	return nil
}

func (i *Info) GetInLibPath() string {
	camDir := i.ExifInfo.CameraMake + " " + i.ExifInfo.CameraModel + " (" + i.ExifInfo.MimeType + ")"
	year := i.ExifInfo.DateTaken.Format("2006")
	date := i.ExifInfo.DateTaken.Format("2006-01-02")
	fileName := i.ExifInfo.DateTaken.Format("2006-01-02_15-04-05") + "_" + i.HashInfo.ShortHash + i.Extension

	return path.Join(camDir, year, date, fileName)
}
