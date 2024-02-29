package scanner

type FileInfo struct {
	Path string

	IsSidecar  bool
	SidecarFor []*FileInfo

	Fields map[string]string
}
