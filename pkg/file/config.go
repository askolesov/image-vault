package file

type Config struct {
	SidecarExtensions      []string
	CameraMakeRemap        map[string]string
	CameraModelRemap       map[string]string
	CameraModelToMakeRemap map[string]string
}

var DefaultConfig = Config{
	SidecarExtensions: []string{".xmp"},
	CameraMakeRemap: map[string]string{
		"SONY": "Sony",
	},
	CameraModelRemap: map[string]string{
		"Canon EOS 5D":   "EOS 5D",
		"Canon EOS 450D": "EOS 450D",
		"Canon EOS 550D": "EOS 550D",
	},
	CameraModelToMakeRemap: map[string]string{
		"EOS 5D":   "Canon",
		"EOS 450D": "Canon",
		"EOS 550D": "Canon",
	},
}
