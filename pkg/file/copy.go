package file

import (
	"io"
	"os"
	"path"
)

func (i *Info) Copy(libPath string, log func(string)) error {
	inLibPath := i.GetInLibPath()
	targetPath := path.Join(libPath, inLibPath)

	// check if file already exists
	if targetInfo, err := os.Stat(targetPath); err == nil {
		if targetInfo.Size() != i.Size {
			log("Overwriting " + i.Path + " to " + targetPath)
			// remove target file if it's smaller than source
			err = os.Remove(targetPath)
			if err != nil {
				return err
			}
		} else {
			log("Skipping " + i.Path + " to " + targetPath)

			// skip if target file is the same size
			return nil
		}
	}

	log("Copying " + i.Path + " to " + targetPath)

	//create directory
	err := os.MkdirAll(path.Dir(targetPath), os.ModePerm)
	if err != nil {
		return err
	}

	// copy file
	srcFile, err := os.Open(i.Path)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}

func (i *Info) GetInLibPath() string {
	camDir := i.ExifInfo.CameraMake + " - " +
		i.ExifInfo.CameraModel + " - " + i.ExifInfo.CameraSerial + " (" + i.ExifInfo.MimeType + ")"
	year := i.ExifInfo.DateTaken.Format("2006")
	date := i.ExifInfo.DateTaken.Format("2006-01-02")
	fileName := i.ExifInfo.DateTaken.Format("2006-01-02_15-04-05") + "_" + i.HashInfo.ShortHash + i.Extension

	return path.Join(camDir, year, date, fileName)
}
