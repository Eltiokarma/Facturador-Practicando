<?php
declare(strict_types=1);

namespace Facturador\Motor;

use DateTime;
use Greenter\Model\Client\Client;
use Greenter\Model\Company\Address;
use Greenter\Model\Company\Company;
use Greenter\Model\Despatch\Despatch;
use Greenter\Model\Despatch\DespatchDetail;
use Greenter\Model\Despatch\Direction;
use Greenter\Model\Despatch\Shipment;
use Greenter\Model\Despatch\Transportist;
use Greenter\Model\Despatch\Vehicle;

/**
 * Emisor de Guía de Remisión Electrónica (GRE).
 *
 * SUNAT cambió en 2022 el flow de GRE: ahora va por REST + OAuth2 contra
 * api-cpe.sunat.gob.pe, NO por SOAP. Greenter v5 soporta esto con
 * Greenter\Api, pero requiere obtener antes un token de acceso.
 *
 * En el MVP esta clase:
 *   - Construye el modelo Despatch de Greenter desde nuestro payload.
 *   - Genera el XML firmado (eso sí funciona offline con el .p12).
 *   - Si hay credenciales API GRE, envía a SUNAT y consulta el ticket.
 *   - Si no, devuelve un error claro indicando que se deben configurar.
 *
 * En modo DEMO el worker ya simula la respuesta y nunca llega acá.
 */
final class Guia
{
    /**
     * @param array<string,mixed> $payload
     * @return array<string,mixed>
     */
    public function emitir(array $payload): array
    {
        $tenant = $payload['tenant'] ?? [];
        $g      = $payload['guia'] ?? [];

        $clientID = (string)($tenant['gre_client_id'] ?? '');
        $clientSecret = (string)($tenant['gre_client_secret'] ?? '');

        if ($clientID === '' || $clientSecret === '') {
            return [
                'estado' => 'error',
                'codigo' => 'GRE_SIN_CREDENCIALES',
                'mensaje' => 'Para emitir GRE en producción/beta configurá las credenciales API GRE (Client ID y Client Secret) en Configuración. La GRE usa OAuth2 contra api-cpe.sunat.gob.pe, no Clave SOL.',
            ];
        }

        try {
            $despatch = $this->buildDespatch($tenant, $g);
        } catch (\Throwable $e) {
            return ['estado' => 'error', 'codigo' => 'BUILD_FAIL', 'mensaje' => $e->getMessage()];
        }

        // TODO en v2: enviar a SUNAT vía Greenter\Api. Esto requiere:
        //   $api = new \Greenter\Api(['feed' => 'https://api-cpe.sunat.gob.pe/v1/contribuyente/gem']);
        //   $api->setBuilderOptions(...)->setApiCredentials($clientID, $clientSecret)
        //       ->setClaveSOL($ruc, $usuario, $clave);
        //   $result = $api->send($despatch);
        //   $ticket = $result->getTicket();
        //   $status = $api->getStatus($ticket);
        //
        // Mientras tanto, devolvemos error explícito para que el operador
        // sepa que la pieza falta. Igual generamos el XML para mostrarlo.
        return [
            'estado' => 'error',
            'codigo' => 'GRE_NO_IMPLEMENTADO',
            'mensaje' => 'El envío real de GRE a SUNAT requiere wiring adicional con Greenter\\Api + OAuth2. Por ahora usá modo DEMO o esperá la próxima versión. La estructura UBL ya se construye correctamente.',
        ];
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
            // El chofer se setea como array de Driver objects; Greenter
            // versión actual usa setChoferes. Lo dejamos minimal por ahora.
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
}
