package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akshithere/task-scheduler/internal/database"
	"github.com/akshithere/task-scheduler/internal/handlers"
	"github.com/akshithere/task-scheduler/internal/middleware"
	"github.com/akshithere/task-scheduler/internal/scheduler"
	"github.com/akshithere/task-scheduler/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	dbConfig := database.LoadConfig()
	if err := database.Connect(dbConfig); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	metrics := scheduler.NewMetrics()

	taskScheduler := scheduler.NewScheduler(metrics)
	if err := taskScheduler.Start(); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}

	taskService := services.NewTaskService()
	taskHandler := handlers.NewTaskHandler(taskService)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(middleware.Logger())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
			"time":   time.Now().UTC(),
		})
	})

	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	api := app.Group("/api/v1")

	api.Post("/tasks", taskHandler.CreateTask)
	api.Get("/tasks", taskHandler.ListTasks)
	api.Get("/tasks/:id", taskHandler.GetTask)
	api.Put("/tasks/:id", taskHandler.UpdateTask)
	api.Delete("/tasks/:id", taskHandler.DeleteTask)
	api.Get("/tasks/:id/results", taskHandler.GetTaskResults)

	api.Get("/results", taskHandler.ListAllResults)

	port := getEnv("PORT", "8080")
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	taskScheduler.Stop()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	if err := database.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	log.Println("Server exited gracefully")
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
