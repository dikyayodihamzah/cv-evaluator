package filectrl

import (
	"github.com/dikyayodihamzah/cv-evaluator/service/filesrv"
	"github.com/gofiber/fiber/v2"
)

type FileController interface {
	Routes(app fiber.App)
}

type fileController struct {
	FileService filesrv.FileService
}

func New(fileService filesrv.FileService) FileController {
	return &fileController{fileService: fileService}
}

func (c *fileController) Routes(app fiber.App) {
	app.Post("/upload", c.upload)
}

func (c *fileController) upload(ctx *fiber.Ctx) error {
	file, err := ctx.FormFile("file")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.fileService.Upload(file)
}
