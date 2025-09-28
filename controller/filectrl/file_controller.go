package filectrl

import (
	"context"
	"mime/multipart"

	"github.com/dikyayodihamzah/cv-evaluator/service/filesrv"
	"github.com/gofiber/fiber/v2"
)

type FileController interface {
	Routes(app fiber.Router)
}

type fileController struct {
	FileService filesrv.FileService
}

func New(fileService filesrv.FileService) FileController {
	return &fileController{FileService: fileService}
}

func (c *fileController) Routes(app fiber.Router) {
	app.Post("/upload", c.upload)
}

func (c *fileController) upload(ctx *fiber.Ctx) error {
	// Parse multipart form
	form, err := ctx.MultipartForm()
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to parse multipart form"})
	}

	files := form.File["files"]
	if len(files) == 0 {
		// Check for single file upload
		file, err := ctx.FormFile("file")
		if err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No files uploaded"})
		}
		files = []*multipart.FileHeader{file}
	}

	// Get upload path from form or default
	path := ctx.FormValue("path")
	if path == "" {
		path = "cv-files"
	}

	// Set context value for uploader (can be extracted from auth later)
	ctxWithUploader := context.WithValue(ctx.Context(), "issuer", "anonymous")

	filePaths, err := c.FileService.Upload(ctxWithUploader, path, files)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{
		"message":    "Files uploaded successfully",
		"file_paths": filePaths,
		"count":      len(filePaths),
	})
}
