<?php
declare(strict_types=1);

namespace Facturador\Motor;

use DateTime;
use Greenter\See;
use Greenter\Model\Company\Address;
use Greenter\Model\Company\Company;
use Greenter\Model\Summary\Summary;
use Greenter\Model\Summary\SummaryDetail;
use Greenter\Ws\Services\SunatEndpoints;

/**
 * Resumen Diario de boletas (RC).
 *
 * Flujo:
 *   1) enviar(): construye el Summary, lo firma, lo envía a SUNAT.
 *      SUNAT devuelve un TICKET (string).
 *   2) consultarTicket(): pregunta a SUNAT por el resultado del ticket.
 *      SUNAT devuelve el CDR final (o "procesando" si todavía no terminó).
 */
final class Resumen
{
    /**
     * @param array<string,mixed> $payload
     * @return array<string,mixed>
     */
    public function enviar(array $payload): array
    {
        $modo = $payload['modo'] ?? 'beta';
        $tenant = $payload['tenant'] ?? [];
        $res = $payload['resumen'] ?? [];

        $see = $this->buildSee($modo, $tenant);
        $summary = $this->buildSummary($tenant, $res);

        try {
            $result = $see->send($summary);
        } catch (\Throwable $e) {
            return [
                'estado' => 'error',
                'codigo' => 'MOTOR_EXCEPTION',
                'mensaje' => $e->getMessage(),
            ];
        }

        $xmlFirmado = $see->getFactory()->getLastXml();

        if ($result === null || !$result->isSuccess()) {
            $err = $result?->getError();
            return [
                'estado' => 'error',
                'codigo' => $err?->getCode() ?? 'UNKNOWN',
                'mensaje' => $err?->getMessage() ?? 'SUNAT rechazó el envío',
            ];
        }

        return [
            'estado' => 'ticket',
            'ticket' => $result->getTicket() ?? '',
            'xml_firmado' => $xmlFirmado ? base64_encode($xmlFirmado) : null,
        ];
    }

    /**
     * @param array<string,mixed> $payload
     * @return array<string,mixed>
     */
    public function consultarTicket(array $payload): array
    {
        $modo = $payload['modo'] ?? 'beta';
        $tenant = $payload['tenant'] ?? [];
        $ticket = (string)($payload['ticket'] ?? '');
        if ($ticket === '') {
            return ['estado' => 'error', 'codigo' => 'BAD_INPUT', 'mensaje' => 'ticket vacío'];
        }

        $see = $this->buildSee($modo, $tenant);

        try {
            $result = $see->getStatus($ticket);
        } catch (\Throwable $e) {
            return [
                'estado' => 'error',
                'codigo' => 'MOTOR_EXCEPTION',
                'mensaje' => $e->getMessage(),
            ];
        }

        if (!$result->isSuccess()) {
            $err = $result->getError();
            // Si el código indica que SUNAT todavía no procesó, devolver "procesando"
            $code = $err?->getCode() ?? '';
            if (in_array($code, ['98', '0098'], true)) {
                return ['estado' => 'procesando'];
            }
            return [
                'estado' => 'rechazado',
                'codigo' => $code,
                'mensaje' => $err?->getMessage() ?? '',
            ];
        }

        $cdr = $result->getCdrResponse();
        $code = $cdr?->getCode() ?? '';
        $obs = $cdr?->getNotes() ?? [];
        $estado = ($code === '0' && empty($obs)) ? 'aceptado' : 'aceptado_con_obs';

        return [
            'estado' => $estado,
            'cdr_zip' => base64_encode($result->getCdrZip() ?? ''),
            'codigo' => $code,
            'mensaje' => $cdr?->getDescription() ?? '',
        ];
    }

    /**
     * @param array<string,mixed> $tenant
     */
    private function buildSee(string $modo, array $tenant): See
    {
        $endpoint = $modo === 'prod' ? SunatEndpoints::FE_PRODUCCION : SunatEndpoints::FE_BETA;
        $cert = trim((string)($tenant['cert_pem'] ?? ''));
        $key  = trim((string)($tenant['cert_key_pem'] ?? ''));
        if ($cert === '' || $key === '') {
            throw new \RuntimeException('cert_pem y cert_key_pem son requeridos');
        }
        $see = new See();
        $see->setCertificate($cert . "\n" . $key . "\n");
        $see->setService($endpoint);
        $see->setCredentials(
            ((string)($tenant['ruc'] ?? '')) . ((string)($tenant['usuario_sol'] ?? '')),
            (string)($tenant['clave_sol'] ?? '')
        );
        return $see;
    }

    /**
     * @param array<string,mixed> $tenant
     * @param array<string,mixed> $res
     */
    private function buildSummary(array $tenant, array $res): Summary
    {
        $company = (new Company())
            ->setRuc((string)$tenant['ruc'])
            ->setRazonSocial((string)$tenant['razon_social'])
            ->setNombreComercial((string)($tenant['nombre_comercial'] ?? $tenant['razon_social']))
            ->setAddress(
                (new Address())
                    ->setUbigueo((string)($tenant['ubigeo'] ?? '150101'))
                    ->setDepartamento((string)($tenant['departamento'] ?? 'LIMA'))
                    ->setProvincia((string)($tenant['provincia'] ?? 'LIMA'))
                    ->setDistrito((string)($tenant['distrito'] ?? 'LIMA'))
                    ->setUrbanizacion('-')
                    ->setDireccion((string)($tenant['direccion_fiscal'] ?? '-'))
                    ->setCodLocal('0000')
            );

        $fechaRef = new DateTime((string)$res['fecha_referencia']);
        $fechaEmi = new DateTime((string)$res['fecha_emision']);

        // El correlativo del RC va sin guion, solo dígitos
        $correlativo = (string)$res['correlativo'];

        $summary = (new Summary())
            ->setCorrelativo($correlativo)
            ->setFecGeneracion($fechaRef)
            ->setFecResumen($fechaEmi)
            ->setCompany($company);

        $details = [];
        foreach ((array)$res['documentos'] as $i => $d) {
            $det = (new SummaryDetail())
                ->setTipoDoc((string)($d['tipo_doc'] ?? '03'))
                ->setSerieNro(((string)$d['serie']) . '-' . ((string)$d['correlativo']))
                ->setEstado('1') // 1: adicionar, 2: modificar, 3: anular
                ->setClienteTipo((string)($d['receptor_tipo_doc'] ?? '1'))
                ->setClienteNro((string)($d['receptor_num_doc'] ?? '-'))
                ->setTotal((float)$d['total'])
                ->setMtoOperGravadas((float)($d['gravado'] ?? 0))
                ->setMtoOperExoneradas((float)($d['exonerado'] ?? 0))
                ->setMtoOperInafectas((float)($d['inafecto'] ?? 0))
                ->setMtoIGV((float)($d['igv'] ?? 0));
            $details[] = $det;
            unset($i);
        }
        $summary->setDetails($details);

        return $summary;
    }
}
