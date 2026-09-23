package main

import (
	"log"
	"os"

	"github.com/Izenberk/url-shortener-api/internal/database"
	"github.com/Izenberk/url-shortener-api/api/routes"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/joho/godotenv"
)

func setupRoutes(app *fiber.App) {
	app.Get("/:url", routes.ResolveURL)
	app.Post("/api/v1", routes.ShortenURL)
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

	setupRoutes(app)

	log.Fatal(app.Listen(os.Getenv("API_PORT")))
}