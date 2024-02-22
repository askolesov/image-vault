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

	Width  int64 `json:"width"`
	Height int64 `json:"height"`

	MimeType  string    `json:"mime_type"`
	DateTaken time.Time `json:"date_taken"`
}

func (i *Info) GetExifInfo(et *exiftool.Exiftool) error {
	metadata := et.ExtractMetadata(i.Path)[0]
	if metadata.Err != nil {
		return metadata.Err
	}

	for k, v := range metadata.Fields {
		vJson, err := json.Marshal(v)
		if err != nil {
			return err
		}

		println(k, string(vJson))
	}

	// DeviceManufacturer (Sony)
	cameraMake, err := metadata.GetString("Make")
	if err != nil {
		cameraMake = "Unknown Make"
	}

	// DeviceModelName (ILCE-6300)
	cameraModel, err := metadata.GetString("Model")
	if err != nil {
		cameraModel = "Unknown Model"
	}

	// DeviceSerialNo (4294967295)
	cameraSerial, err := metadata.GetString("SerialNumber")
	if err != nil {
		cameraSerial = "Unknown Serial Number"
	}

	width, err := metadata.GetInt("ImageWidth")
	if err != nil {
		width = 0
	}

	height, err := metadata.GetInt("ImageHeight")
	if err != nil {
		height = 0
	}

	// MediaCreateDate
	dateTimeStr, err := metadata.GetString("DateTimeOriginal")
	if err != nil {
		dateTimeStr = "1970:01:01 00:00:00"
	}

	dateTime, err := time.Parse("2006:01:02 15:04:05", dateTimeStr)
	if err != nil {
		dateTime = time.Unix(0, 0)
	}

	mimeType, err := metadata.GetString("MIMEType")
	if err != nil {
		mimeType = "Unknown MIME Type"
	}

	i.ExifInfo = &ExifInfo{
		CameraMake:   cameraMake,
		CameraModel:  cameraModel,
		CameraSerial: cameraSerial,

		Width:  width,
		Height: height,

		MimeType:  mimeType,
		DateTaken: dateTime,
	}

	return nil
}

func (i *Info) IsImage() (bool, error) {
	if i.ExifInfo == nil {
		return false, nil
	}

	// width is not 0 and lower mime type starts with "image/"
	isImage := i.ExifInfo.Width != 0 && strings.HasPrefix(strings.ToLower(i.ExifInfo.MimeType), "image/")

	return isImage, nil
}
