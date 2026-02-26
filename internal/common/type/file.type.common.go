package types

import "mime/multipart"

type BufferedFile struct {
	MediaType    string `json:"mediaType" validate:"required"`
	OriginalName string `json:"originalName" validate:"required"`
	Encoding     string `json:"encoding" validate:"required"`
	MimeType     string `json:"mimetype" validate:"required"`
	Size         int    `json:"size" validate:"required"`
	Buffer       []byte `json:"buffer" validate:"required"`
}

type BufferedFiles map[string][]BufferedFile

type UploadFile struct {
	File   multipart.File
	Header *multipart.FileHeader
	Path   string
}

type UploadFilesRes struct {
	OriginalFiles string
	FileName      string
	FileBytes     []byte
	ContentType   string
}
