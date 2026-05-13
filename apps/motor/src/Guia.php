<?php
declare(strict_types=1);

namespace Facturador\Motor;

use DateTime;
use Greenter\Api;
use Greenter\Model\Client\Client;
use Greenter\Model\Company\Address;
use Greenter\Model\Company\Company;
use Greenter\Model\Despatch\Despatch;
use Greenter\Model\Despatch\DespatchDetail;
use Greenter\Model\Despatch\Direction;
use Greenter\Model\Despatch\Driver;
use Greenter\Model\Despatch\Shipment;
use Greenter\Model\Despatch\Transportist;
use Greenter\Model\Despatch\Vehicle;

/**
 * Emisor de Guía de Remisión Electrónica (GRE), tipo 09.
 *
 * SUNAT, desde 2022, exige enviar GRE vía REST + OAuth2 contra
 *   https://api-cpe.sunat.gob.pe/v1  (producción)
 *   https://api-cpe.sunat.gob.pe/v1  (beta — sí, es el mismo host)
 *
 * El XML se sigue firmando con el .p12 del contribuyente (XAdES-BES).
 * Lo que cambia respecto a facturas es:
 *   - Autenticación: OAuth2 client_credentials (no Clave SOL).
 *   - Transporte: HTTP POST JSON (no SOAP).
 *   - Resultado: ticket → consultar.
 *
 * Esta clase implementa el flow completo usando Greenter\Api.
 */
final class Guia
{
    /** Endpoints de SUNAT por modo. */
    private const ENDPOINTS = [
        'prod' => [
            'auth' => 'https://api-seguridad.sunat.gob.pe/v1',
            'cpe'  => 'https://api-cpe.sunat.gob.pe/v1',
        ],
        'beta' => [
            'auth' => 'https://api-seguridad.sunat.gob.pe/v1',
            'cpe'  => 'https://api-cpe.sunat.gob.pe/v1',
            // SUNAT comparte el host para beta y prod; se diferencian por
            // las credenciales que el contribuyente registra en cada
            // ambiente desde Clave SOL.
        ],
    ];

    /**
     * Envía la guía a SUNAT y devuelve el ticket.
     *
     * @param array<string,mixed> $payload
     * @return array<string,mixed>
     */
    public function emitir(array $payload): array
    {
        $modo   = (string)($payload['modo'] ?? 'beta');
        $tenant = (array)($payload['tenant'] ?? []);
        $g      = (array)($payload['guia'] ?? []);

        [$ok, $err] = $this->validarCreds($tenant);
        if (!$ok) return $err;

        try {
            $api = $this->buildApi($modo, $tenant);
            $despatch = $this->buildDespatch($tenant, $g);
            $result = $api->send($despatch);
        } catch (\Throwable $e) {
            return ['estado' => 'error', 'codigo' => 'MOTOR_EXCEPTION', 'mensaje' => $e->getMessage()];
        }

        $xmlFirmado = $api->getLastXml();
        $hash = $this->extraerHash($xmlFirmado ?? '');

        if ($result === null || !$result->isSuccess()) {
            $errObj = $result?->getError();
            return [
                'estado' => 'rechazado',
                'xml_firmado' => $xmlFirmado ? base64_encode($xmlFirmado) : null,
                'hash_cpe' => $hash,
                'codigo' => $errObj?->getCode() ?? 'UNKNOWN',
                'mensaje' => $errObj?->getMessage() ?? 'SUNAT rechazó la GRE',
            ];
        }

        // En Greenter\Api, send() devuelve BillResult con ticket.
        $ticket = '';
        if (method_exists($result, 'getTicket')) {
            $ticket = (string)$result->getTicket();
        }
        if ($ticket === '') {
            return [
                'estado' => 'error',
                'codigo' => 'SIN_TICKET',
                'mensaje' => 'SUNAT no devolvió ticket para la GRE',
            ];
        }

        return [
            'estado' => 'ticket',
            'ticket' => $ticket,
            'xml_firmado' => $xmlFirmado ? base64_encode($xmlFirmado) : null,
            'hash_cpe' => $hash,
        ];
    }

    /**
     * Consulta el resultado de un ticket de GRE.
     *
     * @param array<string,mixed> $payload
     * @return array<string,mixed>
     */
    public function consultarTicket(array $payload): array
    {
        $modo   = (string)($payload['modo'] ?? 'beta');
        $tenant = (array)($payload['tenant'] ?? []);
        $ticket = (string)($payload['ticket'] ?? '');

        [$ok, $err] = $this->validarCreds($tenant);
        if (!$ok) return $err;
        if ($ticket === '') {
            return ['estado' => 'error', 'codigo' => 'BAD_INPUT', 'mensaje' => 'ticket vacío'];
        }

        try {
            $api = $this->buildApi($modo, $tenant);
            $status = $api->getStatus($ticket);
        } catch (\Throwable $e) {
            return ['estado' => 'error', 'codigo' => 'MOTOR_EXCEPTION', 'mensaje' => $e->getMessage()];
        }

        if (!$status->isSuccess()) {
            $errObj = $status->getError();
            $code = $errObj?->getCode() ?? '';
            // 98 = procesando (SUNAT todavía no terminó).
            if (in_array($code, ['98', '0098'], true)) {
                return ['estado' => 'procesando'];
            }
            return [
                'estado' => 'rechazado',
                'codigo' => $code,
                'mensaje' => $errObj?->getMessage() ?? 'SUNAT rechazó',
            ];
        }

        $cdr = $status->getCdrResponse();
        $code = $cdr?->getCode() ?? '';
        $obs = $cdr?->getNotes() ?? [];
        $estado = ($code === '0' && empty($obs)) ? 'aceptado' : 'aceptado_con_obs';

        return [
            'estado'  => $estado,
            'cdr_zip' => base64_encode($status->getCdrZip() ?? ''),
            'codigo'  => $code,
            'mensaje' => $cdr?->getDescription() ?? '',
        ];
    }

    /**
     * @param array<string,mixed> $tenant
     * @return array{0:bool,1:array<string,mixed>}
     */
    private function validarCreds(array $tenant): array
    {
        $cert = trim((string)($tenant['cert_pem'] ?? ''));
        $key  = trim((string)($tenant['cert_key_pem'] ?? ''));
        $clientID = (string)($tenant['gre_client_id'] ?? '');
        $secret = (string)($tenant['gre_client_secret'] ?? '');
        if ($cert === '' || $key === '') {
            return [false, ['estado' => 'error', 'codigo' => 'NO_CERT',
                'mensaje' => 'GRE requiere certificado .p12 — subilo desde Configuración']];
        }
        if ($clientID === '' || $secret === '') {
            return [false, ['estado' => 'error', 'codigo' => 'NO_API_CREDS',
                'mensaje' => 'GRE requiere Client ID + Client Secret API SUNAT. Obtenelos en Clave SOL → Cliente API.']];
        }
        return [true, []];
    }

    /**
     * @param array<string,mixed> $tenant
     */
    private function buildApi(string $modo, array $tenant): Api
    {
        $endpoints = self::ENDPOINTS[$modo] ?? self::ENDPOINTS['beta'];
        $api = new Api($endpoints);
        $api->setBuilderOptions(['strict_variables' => true]);
        $api->setApiCredentials(
            (string)$tenant['gre_client_id'],
            (string)$tenant['gre_client_secret']
        );
        // Algunas versiones de Greenter\Api también requieren Clave SOL
        // para el flow de GRE (depende del ambiente). Pasamos lo que tenga.
        if (!empty($tenant['usuario_sol']) && !empty($tenant['clave_sol'])) {
            $api->setClaveSOL(
                (string)$tenant['ruc'],
                (string)$tenant['usuario_sol'],
                (string)$tenant['clave_sol']
            );
        }
        $cert = trim((string)$tenant['cert_pem']) . "\n" . trim((string)$tenant['cert_key_pem']) . "\n";
        $api->setCertificate($cert);
        return $api;
    }

    /**
     * @param array<string,mixed> $tenant
     * @param array<string,mixed> $g
     */
    private function buildDespatch(array $tenant, array $g): Despatch
    {
        $company = (new Company())
            ->setRuc((string)$tenant['ruc'])
            ->setRazonSocial((string)$tenant['razon_social'])
            ->setNombreComercial((string)($tenant['nombre_comercial'] ?? $tenant['razon_social']))
            ->setAddress(
                (new Address())
                    ->setUbigueo((string)($tenant['ubigeo'] ?? '150101'))
                    ->setDepartamento('-')
                    ->setProvincia('-')
                    ->setDistrito('-')
                    ->setUrbanizacion('-')
                    ->setDireccion((string)($tenant['direccion_fiscal'] ?? '-'))
                    ->setCodLocal('0000')
            );

        $dest = (array)$g['destinatario'];
        $destinatario = (new Client())
            ->setTipoDoc((string)$dest['tipo_doc'])
            ->setNumDoc((string)$dest['num_doc'])
            ->setRznSocial((string)$dest['razon_social']);

        $partida = (array)$g['punto_partida'];
        $llegada = (array)$g['punto_llegada'];

        $envio = (new Shipment())
            ->setCodTraslado((string)$g['motivo'])
            ->setModTraslado((string)$g['modalidad'])
            ->setFecTraslado(new DateTime((string)$g['fecha_traslado']))
            ->setPesoTotal((float)$g['peso_total_kg'])
            ->setUndPesoTotal('KGM')
            ->setNumBultos((int)($g['num_bultos'] ?? 1))
            ->setLlegada(
                (new Direction())
                    ->setUbigueo((string)$llegada['ubigeo'])
                    ->setDireccion((string)$llegada['direccion'])
            )
            ->setPartida(
                (new Direction())
                    ->setUbigueo((string)$partida['ubigeo'])
                    ->setDireccion((string)$partida['direccion'])
            );

        if (($g['modalidad'] ?? '') === '01' && !empty($g['transportista'])) {
            $t = (array)$g['transportista'];
            $envio->setTransportista(
                (new Transportist())
                    ->setTipoDoc('6')
                    ->setNumDoc((string)$t['ruc'])
                    ->setRznSocial((string)($t['razon_social'] ?? ''))
            );
        } elseif (($g['modalidad'] ?? '') === '02' && !empty($g['transportista'])) {
            $t = (array)$g['transportista'];
            $envio->setVehiculo(
                (new Vehicle())->setPlaca((string)$t['placa_vehiculo'])
            );
            if (!empty($t['doc_chofer']) && !empty($t['nombre_chofer'])) {
                $chofer = (new Driver())
                    ->setTipoDoc('1')
                    ->setNroDoc((string)$t['doc_chofer'])
                    ->setNombres((string)$t['nombre_chofer'])
                    ->setApellidos('')
                    ->setLicencia((string)($t['licencia_chofer'] ?? ''))
                    ->setTipo('Principal');
                $envio->setChoferes([$chofer]);
            }
        }

        $despatch = (new Despatch())
            ->setVersion('2022')
            ->setTipoDoc('09')
            ->setSerie((string)$g['serie'])
            ->setCorrelativo((string)$g['correlativo'])
            ->setFechaEmision(new DateTime((string)$g['fecha_emision']))
            ->setCompany($company)
            ->setDestinatario($destinatario)
            ->setEnvio($envio);

        if (!empty($g['observaciones'])) {
            $despatch->setObservacion((string)$g['observaciones']);
        }

        $details = [];
        foreach ((array)$g['items'] as $it) {
            $details[] = (new DespatchDetail())
                ->setCantidad((float)$it['cantidad'])
                ->setUnidad((string)($it['unidad'] ?? 'NIU'))
                ->setDescripcion((string)$it['descripcion'])
                ->setCodigo((string)($it['codigo'] ?? ''));
        }
        $despatch->setDetails($details);
        return $despatch;
    }

    private function extraerHash(string $xml): string
    {
        if ($xml === '') return '';
        $doc = new \DOMDocument();
        if (!@$doc->loadXML($xml)) return '';
        $nodes = $doc->getElementsByTagNameNS('http://www.w3.org/2000/09/xmldsig#', 'DigestValue');
        return $nodes->length > 0 ? trim($nodes->item(0)->nodeValue ?? '') : '';
    }
}
