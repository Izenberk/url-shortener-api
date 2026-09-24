package routes

import (
	"log"

	"github.com/Izenberk/url-shortener-api/internal/database"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

// ResolveURL resolves a short URL to its original and redirects the client.
func ResolveURL(c fiber.Ctx) error {
	url := c.Params("url")
	ctx := c.Context()

	value, err := database.DB0.Get(ctx, url).Result()
	if err == redis.Nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "short no found",
		})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "cannot connect to DB",
		})
	}

	if err := database.DB1.Incr(ctx, "counter").Err(); err != nil {
		log.Printf("warn: failed to increment redirect counter: %v", err)
	}

	c.Set("Cache-Control", "no-store")
	return c.Redirect().Status(fiber.StatusFound).To(value)
}
