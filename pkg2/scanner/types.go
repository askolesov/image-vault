package scanner

type FileInfo struct {
	Path string

	IsSidecar  bool
	SidecarFor []*FileInfo
}
