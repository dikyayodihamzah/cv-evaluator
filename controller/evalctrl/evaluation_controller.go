package evalctrl

import (
	"github.com/dikyayodihamzah/cv-evaluator/pkg/model"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/model/cvweb"
	"github.com/dikyayodihamzah/cv-evaluator/service/evalsrv"
	"github.com/gofiber/fiber/v2"
)

type EvaluationController interface {
	Routes(app fiber.Router)
}

type evaluationController struct {
	EvaluationService evalsrv.EvaluationService
}

func New(evaluationService evalsrv.EvaluationService) EvaluationController {
	return &evaluationController{EvaluationService: evaluationService}
}

func (c *evaluationController) Routes(app fiber.Router) {
	app.Post("/evaluate", c.evaluate)
	app.Get("/result/:id", c.getResult)
}

func (c *evaluationController) evaluate(ctx *fiber.Ctx) error {
	var req cvweb.EvaluationRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(model.BaseResponse{
			Status:  "error",
			Message: "Invalid request body",
		})
	}

	jobID, err := c.EvaluationService.StartEvaluation(req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(model.BaseResponse{
			Status:  "error",
			Message: "Failed to start evaluation",
		})
	}

	return ctx.JSON(model.BaseResponse{
		ID:     jobID,
		Status: "queued",
	})
}

func (c *evaluationController) getResult(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(model.BaseResponse{
			Status:  "error",
			Message: "Job ID is required",
		})
	}

	result, err := c.EvaluationService.GetResult(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(model.BaseResponse{
			Status:  "error",
			Message: "Job not found",
		})
	}

	return ctx.JSON(result)
}
