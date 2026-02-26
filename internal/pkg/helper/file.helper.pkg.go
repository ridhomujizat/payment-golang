package helper

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	_type "go-boilerplate/internal/common/type"
	"path/filepath"

	"github.com/google/uuid"
)

func PrepareFileUploadPayload(p _type.UploadFile) (*_type.UploadFilesRes, error) {
	// Seek ulang ke awal (WAJIB!)
	if seeker, ok := p.File.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to seek: %w", err)
		}
	}

	// Baca isi file ke buffer
	buf := bytes.NewBuffer(nil)
	if _, err := buf.ReadFrom(p.File); err != nil {
		return nil, fmt.Errorf("failed to read from file: %w", err)
	}

	// // Log panjang file
	// fmt.Println("File Size:", buf.Len())

	// Tentukan extension dan path upload
	ext := filepath.Ext(p.Header.Filename)
	folder := "uploads/"
	if p.Path != "" {
		folder = p.Path + "/"
	}
	fileName := folder + uuid.New().String() + ext

	// Tentukan content-type
	contentType := p.Header.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(buf.Bytes())
	}

	// // Debug log
	// fmt.Println("Content-Type:", contentType)
	// fmt.Println("Generated FileName:", fileName)

	// // Simpan lokal untuk debug
	// _ = os.WriteFile("test-debug.pdf", buf.Bytes(), 0644)

	return &_type.UploadFilesRes{
		OriginalFiles: fileName,
		FileName:      p.Header.Filename,
		FileBytes:     buf.Bytes(),
		ContentType:   contentType,
	}, nil
}
