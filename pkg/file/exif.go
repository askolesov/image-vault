package file

import (
	"encoding/json"
	"github.com/barasher/go-exiftool"
	"strings"
	"time"
)

type ExifInfo struct {
	CameraMake   string `json:"camera_make"`
	CameraModel  string `json:"camera_model"`
	CameraSerial string `json:"camera_serial"`

	MimeType  string    `json:"mime_type"`
	DateTaken time.Time `json:"date_taken"`
}

func (i *Info) GetExifInfo(et *exiftool.Exiftool, debug bool) error {
	metadata := et.ExtractMetadata(i.Path)[0]
	if metadata.Err != nil {
		return metadata.Err
	}

	if debug {
		for k, v := range metadata.Fields {
			vJson, err := json.Marshal(v)
			if err != nil {
				return err
			}

			println(k, string(vJson))
		}
	}

	i.ExifInfo = &ExifInfo{
		CameraMake:   GetCameraMake(metadata),
		CameraModel:  GetCameraModel(metadata),
		CameraSerial: GetCameraSerial(metadata),

		MimeType:  GetMimeType(metadata),
		DateTaken: GetDateTime(metadata),
	}

	if newMake, ok := DefaultConfig.CameraModelToMakeRemap[i.ExifInfo.CameraModel]; ok {
		i.ExifInfo.CameraMake = newMake
	}

	return nil
}

func GetCameraMake(m exiftool.FileMetadata) string {
	result := GetVal(m, []string{"Make", "DeviceManufacturer"}, "NoMake")
	return RemapVal(result, DefaultConfig.CameraMakeRemap)
}

func GetCameraModel(m exiftool.FileMetadata) string {
	result := GetVal(m, []string{"Model", "DeviceModelName"}, "NoModel")
	return RemapVal(result, DefaultConfig.CameraModelRemap)
}

func GetCameraSerial(m exiftool.FileMetadata) string {
	return GetVal(m, []string{"SerialNumber", "DeviceSerialNo"}, "NoSerial")
}

func GetMimeType(m exiftool.FileMetadata) string {
	result := GetVal(m, []string{"MIMEType"}, "NoMIME")

	// take part before the slash
	parts := strings.Split(result, "/")
	if len(parts) > 0 {
		result = strings.ToLower(parts[0])
	}

	return result
}

func GetDateTime(m exiftool.FileMetadata) time.Time {
	dateTimeStr := GetVal(m, []string{"DateTimeOriginal", "MediaCreateDate"}, "1970:01:01 00:00:00")

	dateTime, err := time.Parse("2006:01:02 15:04:05", dateTimeStr)
	if err != nil {
		dateTime = time.Unix(0, 0)
	}

	dateTime = dateTime.UTC()

	return dateTime
}

func GetVal(m exiftool.FileMetadata, tags []string, defVal string) string {
	result := ""
	for _, tag := range tags {
		if result != "" {
			break
		}

		val, err := m.GetString(tag)
		if err == nil {
			result = val
		}
	}

	if result == "" {
		result = defVal
	}

	return result
}

func RemapVal(val string, templates map[string]string) string {
	if newVal, ok := templates[val]; ok {
		return newVal
	}

	return val
}
