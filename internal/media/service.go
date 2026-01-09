// internal/media/service.go
package media

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	baseDir string
	baseURL string
}

func NewService(baseDir string, baseURL string) *Service {
	return &Service{baseDir: baseDir, baseURL: baseURL}
}

type UploadResult struct {
	HeadshotPath pgtype.Text
	GovIDPath    pgtype.Text
	PassportPath pgtype.Text
}

type PresignedResponse struct {
	Key         string `json:"key"`
	UploadURL   string `json:"upload_url"`
	DownloadURL string `json:"download_url"`
}

func (s *Service) GenerateMockPresignedURL(userID string, fileType string) PresignedResponse {
	uniqueName := fmt.Sprintf("%s_%s.jpg", fileType, uuid.New().String())
	storageKey := filepath.Join(userID, uniqueName)

	return PresignedResponse{
		Key:         storageKey,
		UploadURL:   fmt.Sprintf("%s/api/v1/media/upload/%s", s.baseURL, storageKey),
		DownloadURL: fmt.Sprintf("%s/store/media/%s", s.baseURL, storageKey),
	}
}

func (s *Service) SaveMockUpload(storageKey string, fileContent io.Reader) error {
	fullPath := filepath.Join(s.baseDir, storageKey)
	if err := os.MkdirAll(filepath.Dir(fullPath), os.ModePerm); err != nil {
		return err
	}
	dst, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, fileContent)
	return err
}
