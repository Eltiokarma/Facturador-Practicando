<?php
declare(strict_types=1);

require __DIR__ . '/../vendor/autoload.php';

use Facturador\Motor\Emisor;
use Facturador\Motor\Guia;
use Facturador\Motor\Nota;
use Facturador\Motor\Resumen;
use Slim\Factory\AppFactory;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;

error_reporting(E_ALL);
ini_set('display_errors', '0');

$app = AppFactory::create();
$app->addBodyParsingMiddleware();
$app->addRoutingMiddleware();
$errorMw = $app->addErrorMiddleware(true, true, true);

$app->get('/health', function (Request $request, Response $response): Response {
    $payload = json_encode([
        'status' => 'ok',
        'sunat_mode' => getenv('SUNAT_MODE') ?: 'beta',
        'time' => gmdate('c'),
    ]);
    $response->getBody()->write($payload);
    return $response->withHeader('Content-Type', 'application/json');
});

$app->post('/emitir', function (Request $request, Response $response): Response {
    $payload = $request->getParsedBody();
    if (!is_array($payload)) {
        $response->getBody()->write(json_encode([
            'estado' => 'error',
            'codigo' => 'BAD_PAYLOAD',
            'mensaje' => 'JSON inválido',
        ]));
        return $response->withStatus(400)->withHeader('Content-Type', 'application/json');
    }

    $tipo = (string)(($payload['comprobante'] ?? [])['tipo'] ?? '');
    try {
        if ($tipo === '07' || $tipo === '08') {
            $resultado = (new Nota())->emitir($payload);
        } else {
            $resultado = (new Emisor())->emitir($payload);
        }
    } catch (\Throwable $e) {
        $resultado = [
            'estado' => 'error',
            'codigo' => 'MOTOR_EXCEPTION',
            'mensaje' => $e->getMessage(),
        ];
    }

    $status = match ($resultado['estado'] ?? 'error') {
        'aceptado', 'aceptado_con_obs', 'rechazado' => 200,
        default => 500,
    };

    $response->getBody()->write((string)json_encode($resultado, JSON_UNESCAPED_UNICODE));
    return $response->withStatus($status)->withHeader('Content-Type', 'application/json');
});

$app->post('/emitir-guia', function (Request $request, Response $response): Response {
    $payload = $request->getParsedBody();
    if (!is_array($payload)) {
        $response->getBody()->write(json_encode([
            'estado' => 'error',
            'codigo' => 'BAD_PAYLOAD',
            'mensaje' => 'JSON inválido',
        ]));
        return $response->withStatus(400)->withHeader('Content-Type', 'application/json');
    }
    try {
        $r = (new Guia())->emitir($payload);
    } catch (\Throwable $e) {
        $r = ['estado' => 'error', 'codigo' => 'MOTOR_EXCEPTION', 'mensaje' => $e->getMessage()];
    }
    $status = ($r['estado'] ?? 'error') === 'error' ? 500 : 200;
    $response->getBody()->write((string)json_encode($r, JSON_UNESCAPED_UNICODE));
    return $response->withStatus($status)->withHeader('Content-Type', 'application/json');
});

$app->post('/resumen', function (Request $request, Response $response): Response {
    $payload = $request->getParsedBody();
    if (!is_array($payload)) {
        $response->getBody()->write(json_encode([
            'estado' => 'error',
            'codigo' => 'BAD_PAYLOAD',
            'mensaje' => 'JSON inválido',
        ]));
        return $response->withStatus(400)->withHeader('Content-Type', 'application/json');
    }
    try {
        $r = (new Resumen())->enviar($payload);
    } catch (\Throwable $e) {
        $r = ['estado' => 'error', 'codigo' => 'MOTOR_EXCEPTION', 'mensaje' => $e->getMessage()];
    }
    $status = ($r['estado'] ?? 'error') === 'error' ? 500 : 200;
    $response->getBody()->write((string)json_encode($r, JSON_UNESCAPED_UNICODE));
    return $response->withStatus($status)->withHeader('Content-Type', 'application/json');
});

$app->post('/consulta-ticket', function (Request $request, Response $response): Response {
    $payload = $request->getParsedBody();
    if (!is_array($payload)) {
        $response->getBody()->write(json_encode([
            'estado' => 'error',
            'codigo' => 'BAD_PAYLOAD',
            'mensaje' => 'JSON inválido',
        ]));
        return $response->withStatus(400)->withHeader('Content-Type', 'application/json');
    }
    try {
        $r = (new Resumen())->consultarTicket($payload);
    } catch (\Throwable $e) {
        $r = ['estado' => 'error', 'codigo' => 'MOTOR_EXCEPTION', 'mensaje' => $e->getMessage()];
    }
    $status = ($r['estado'] ?? 'error') === 'error' ? 500 : 200;
    $response->getBody()->write((string)json_encode($r, JSON_UNESCAPED_UNICODE));
    return $response->withStatus($status)->withHeader('Content-Type', 'application/json');
});

$app->run();
