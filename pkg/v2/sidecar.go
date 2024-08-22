package v2

type FileInfoSidecar struct {
	Path string

	IsSidecar  bool
	SidecarFor []*FileInfoSidecar
}

func LinkSidecars(
	files []*FileInfoSidecar,
) {
	for i, file := range files {
		for j, other := range files {
			if i == j {
				continue
			}

			if file.Path == other.Path {
				file.SidecarFor = append(file.SidecarFor, other)
				other.IsSidecar = true
			}
		}
	}
}
