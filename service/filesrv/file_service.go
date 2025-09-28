package filesrv

import "mime/multipart"

type FileService interface {
	Upload(file *multipart.FileHeader) (string, error)
}

type fileService struct {
}

func New() FileService {
	return &fileService{}
}

func (f *fileService) Upload(file *multipart.FileHeader) (string, error) {
	return "", nil
}
