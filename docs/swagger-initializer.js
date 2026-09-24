window.addEventListener('DOMContentLoaded', () => {
  window.ui = SwaggerUIBundle({
    url: '/docs/openapi.yaml',
    dom_id: '#swagger-ui',
    deepLinking: true,
    validatorUrl: null,
    supportedSubmitMethods: ['get', 'post'],
    docExpansion: 'list',
    defaultModelsExpandDepth: -1,
    defaultModelRendering: 'model',
  });
});
