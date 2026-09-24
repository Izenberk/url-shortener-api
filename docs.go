package main

import (
	"embed"

	"github.com/gofiber/fiber/v3"
)

// Embed the specification and UI so the Docker runtime needs only the Go binary.
//go:embed openapi.yaml docs/index.html docs/swagger-initializer.js docs/vendor/*
var documentation embed.FS

func setupDocs(app *fiber.App) {
	serve := func(path, contentType string) fiber.Handler {
		data, err := documentation.ReadFile(path)
		if err != nil {
			panic(err) // Embedded files are required at build time.
		}
		return func(c fiber.Ctx) error {
			c.Set(fiber.HeaderContentType, contentType)
			return c.Send(data)
		}
	}

	app.Get("/docs", serve("docs/index.html", "text/html; charset=utf-8"))
	app.Get("/docs/openapi.yaml", serve("openapi.yaml", "application/yaml; charset=utf-8"))
	app.Get("/docs/swagger-initializer.js", serve("docs/swagger-initializer.js", "text/javascript; charset=utf-8"))
	app.Get("/docs/swagger-ui.css", serve("docs/vendor/swagger-ui.css", "text/css; charset=utf-8"))
	app.Get("/docs/swagger-ui-bundle.js", serve("docs/vendor/swagger-ui-bundle.js", "text/javascript; charset=utf-8"))
	app.Get("/docs/licenses/LICENSE", serve("docs/vendor/LICENSE", "text/plain; charset=utf-8"))
	app.Get("/docs/licenses/NOTICE", serve("docs/vendor/NOTICE", "text/plain; charset=utf-8"))
	app.Get("/docs/licenses/bundle", serve("docs/vendor/swagger-ui-bundle.js.LICENSE.txt", "text/plain; charset=utf-8"))
}
