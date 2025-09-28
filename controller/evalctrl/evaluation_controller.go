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

	// Validate request - CV file is required
	if req.CVFile == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(model.BaseResponse{
			Status:  "error",
			Message: "CV file is required",
		})
	}

	// Check if job description is provided either as text or file
	if req.JobDescription == "" && req.JobDescriptionFile == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(model.BaseResponse{
			Status:  "error",
			Message: "Either job_description or job_description_file is required",
		})
	}

	// Project file is optional - if not provided, project content will be extracted from CV file
	// This provides flexibility for users to either:
	// 1. Use a single CV file that contains both CV and project information
	// 2. Use separate CV and project files for more detailed evaluation

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
