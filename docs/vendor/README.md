# Vendored Swagger UI

Source: official npm package `swagger-ui-dist@5.33.0`.
Upstream: https://github.com/swagger-api/swagger-ui
Package: https://www.npmjs.com/package/swagger-ui-dist/v/5.33.0

The bundle and stylesheet are copied unchanged. LICENSE, NOTICE, and
swagger-ui-bundle.js.LICENSE.txt are retained and exposed by the documentation
page. Source maps and optional OAuth/standalone assets are not required by this UI.

These files are embedded in the Go binary. npm is not needed to build or run
the API. To upgrade, obtain a reviewed, explicit package version, replace both
assets and the license files together, update the version here and in index.html,
then rerun the documentation tests and rebuild the API.
