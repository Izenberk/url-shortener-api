package main

import (
	"log"
	"os"

	"github.com/Izenberk/url-shortener-api/internal/database"
	"github.com/Izenberk/url-shortener-api/internal/handlers"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/joho/godotenv"
)

func setupRoutes(app *fiber.App) {
	// Register documentation before the dynamic short-code route.
	setupDocs(app)
	app.Get("/:url", handlers.ResolveURL)
	app.Post("/api/v1", handlers.ShortenURL)
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("warn: .env file not found, relying on environment variables: %v", err)
	}

	if err := database.Init(); err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}

	app := fiber.New()
	app.Use(logger.New())
	app.Use(cors.New())

	setupRoutes(app)

	log.Fatal(app.Listen(os.Getenv("API_PORT")))
}
