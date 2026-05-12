<?php
declare(strict_types=1);

require __DIR__ . '/../vendor/autoload.php';

use Slim\Factory\AppFactory;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;

$app = AppFactory::create();
$app->addBodyParsingMiddleware();
$app->addRoutingMiddleware();
$app->addErrorMiddleware(true, true, true);

$app->get('/health', function (Request $request, Response $response): Response {
    $payload = json_encode([
        'status' => 'ok',
        'sunat_mode' => getenv('SUNAT_MODE') ?: 'beta',
        'time' => gmdate('c'),
    ]);
    $response->getBody()->write($payload);
    return $response->withHeader('Content-Type', 'application/json');
});

// POST /emitir
// Body: JSON con el comprobante (estructura definida en CONTRACT.md)
// Response: { xml_firmado, cdr_zip, hash_cpe, codigo, mensaje, estado }
$app->post('/emitir', function (Request $request, Response $response): Response {
    $payload = json_encode([
        'error' => 'not_implemented',
        'message' => 'El motor está scaffoldeado pero la integración con Greenter aún no se conectó. Ver apps/motor/CONTRACT.md.',
    ]);
    $response->getBody()->write($payload);
    return $response
        ->withStatus(501)
        ->withHeader('Content-Type', 'application/json');
});

$app->run();
