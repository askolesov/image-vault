package config

type Config struct {
	// tag
	CameraMake   Tag
	CameraModel  Tag
	CameraSerial Tag
	DateTime     Tag

	// tag to tag remaps
	CameraModelToMakeRemap map[string]string

	// SidecarExtensions is a list of file extensions that are considered sidecar files.
	// Those files are copied to the library with the same name as the main file.
	SidecarExtensions []string
}

type Tag struct {
	ExifTags  []string          // list of exif tags to search for value
	Remapping map[string]string // remap values to new values
	Default   string            // default value if tag is not found
}

var DefaultConfig = Config{
	CameraMake: Tag{
		ExifTags: []string{"Make", "DeviceManufacturer"},
		Remapping: map[string]string{
			"SONY": "Sony",
		},
		Default: "NoMake",
	},
	CameraModel: Tag{
		ExifTags: []string{"Model", "DeviceModelName"},
		Remapping: map[string]string{
			"Canon EOS 5D":   "EOS 5D",
			"Canon EOS 450D": "EOS 450D",
			"Canon EOS 550D": "EOS 550D",
		},
		Default: "NoModel",
	},
	CameraSerial: Tag{
		ExifTags: []string{"SerialNumber", "DeviceSerialNo"},
		Default:  "NoSerial",
	},
	DateTime: Tag{
		ExifTags: []string{"DateTimeOriginal", "MediaCreateDate"},
		Default:  "1970:01:01 00:00:00",
	},

	CameraModelToMakeRemap: map[string]string{
		"EOS 5D":   "Canon",
		"EOS 450D": "Canon",
		"EOS 550D": "Canon",
	},

	SidecarExtensions: []string{".xmp"},
}
