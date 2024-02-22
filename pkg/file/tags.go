package file

type Category string

const (
	Image Category = "image"
	Video Category = "video"
)

var Extensions = map[string]Category{
	".jpg":  Image,
	".jpeg": Image,
	".mp4":  Video,
}

type TagsInfo struct {
	Category  Category `json:"category"`
	Supported bool     `json:"supported"`
}

func (i *Info) GetTagsInfo() {
	category, supported := Extensions[i.Extension]

	i.TagsInfo = &TagsInfo{
		Category:  category,
		Supported: supported,
	}
}
