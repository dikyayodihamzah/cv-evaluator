package main

import (
	"log"

	"github.com/dikyayodihamzah/cv-evaluator/controller/evalctrl"
	"github.com/dikyayodihamzah/cv-evaluator/controller/filectrl"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/config/minioclient"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/env"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/exception"
	"github.com/dikyayodihamzah/cv-evaluator/service/evalsrv"
	"github.com/dikyayodihamzah/cv-evaluator/service/filesrv"
	"github.com/dikyayodihamzah/cv-evaluator/service/llmsrv"
	"github.com/dikyayodihamzah/cv-evaluator/service/ragsrv"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: exception.Handler,
	})

	// Add middleware
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Initialize MinIO client
	minioClient := minioclient.New()

	// Initialize services
	ragService := ragsrv.New()
	if err := ragService.Initialize(); err != nil {
		log.Fatal("Failed to initialize RAG service:", err)
	}

	fileService := filesrv.New(minioClient)
	llmService := llmsrv.New(ragService)
	evaluationService := evalsrv.New(fileService, llmService)

	// Initialize controllers
	fileController := filectrl.New(fileService)
	evaluationController := evalctrl.New(evaluationService)

	// Setup routes
	api := app.Group("/api/v1")

	// File upload routes
	fileController.Routes(api)

	// Evaluation routes
	evaluationController.Routes(api)

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "cv-evaluator",
			"version": "1.0.0",
		})
	})

	// API documentation endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"service": "CV Evaluator API",
			"version": "1.0.0",
			"endpoints": fiber.Map{
				"health":     "GET /health",
				"upload":     "POST /api/v1/upload",
				"evaluate":   "POST /api/v1/evaluate",
				"get_result": "GET /api/v1/result/{id}",
			},
			"description": "Backend service for evaluating candidate CVs and project reports using AI workflows",
		})
	})

	// Start server
	port := ":" + env.GetString("PORT", "8080")
	log.Printf("🚀 Server starting on port %s", port)
	log.Printf("📚 API Documentation: http://localhost%s", port)
	log.Printf("❤️  Health Check: http://localhost%s/health", port)
	log.Printf("🔧 Environment: %s", env.GetString("ENVIRONMENT"))

	log.Fatal(app.Listen(port))
}
