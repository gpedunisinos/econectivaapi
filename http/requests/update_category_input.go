package requests

type UpdateCategoryInput struct {
	Name         string `json:"name" binding:"required,max=255"`
	Type         string `json:"type" binding:"required,max=100"`
	ColorHex     string `json:"color_hex" binding:"required,len=7"`
	IconCode     string `json:"icon_code" binding:"max=255"`
	Description  string `json:"description"`
	MetadataTags string `json:"metadata_tags" binding:"required"`
}
