<?php
declare(strict_types=1);

namespace Facturador\Motor;

use DateTime;
use Greenter\See;
use Greenter\Model\Company\Address;
use Greenter\Model\Company\Company;
use Greenter\Model\Voided\Voided;
use Greenter\Model\Voided\VoidedDetail;
use Greenter\Ws\Services\SunatEndpoints;

/**
 * Comunicación de baja (anulación de CPE ya aceptados).
 *
 * Documento UBL VoidedDocuments. Va por el mismo endpoint SOAP que las
 * facturas (no por la API REST de GRE). El flow es ticket → consulta
 * igual que el resumen diario.
 */
final class Anulacion
{
    /**
     * @param array<string,mixed> $payload
     * @return array<string,mixed>
     */
    public function enviar(array $payload): array
    {
        $modo   = $payload['modo'] ?? 'beta';
        $tenant = $payload['tenant'] ?? [];
        $a      = $payload['anulacion'] ?? [];

        $see = $this->buildSee($modo, $tenant);
        $voided = $this->buildVoided($tenant, $a);

        try {
            $result = $see->send($voided);
        } catch (\Throwable $e) {
            return ['estado' => 'error', 'codigo' => 'MOTOR_EXCEPTION', 'mensaje' => $e->getMessage()];
        }

        $xmlFirmado = $see->getFactory()->getLastXml();

        if ($result === null || !$result->isSuccess()) {
            $err = $result?->getError();
            return [
                'estado' => 'error',
                'codigo' => $err?->getCode() ?? 'UNKNOWN',
                'mensaje' => $err?->getMessage() ?? 'SUNAT rechazó',
            ];
        }

        return [
            'estado' => 'ticket',
            'ticket' => $result->getTicket() ?? '',
            'xml_firmado' => $xmlFirmado ? base64_encode($xmlFirmado) : null,
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
     * @param array<string,mixed> $a
     */
    private function buildVoided(array $tenant, array $a): Voided
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

        $fechaRef = new DateTime((string)$a['fecha_referencia']);
        $fechaEmi = new DateTime((string)$a['fecha_emision']);
        $correlativo = (string)$a['correlativo'];

        $voided = (new Voided())
            ->setCorrelativo($correlativo)
            ->setFecGeneracion($fechaRef)
            ->setFecComunicacion($fechaEmi)
            ->setCompany($company);

        $details = [];
        foreach ((array)$a['documentos'] as $d) {
            $details[] = (new VoidedDetail())
                ->setTipoDoc((string)$d['tipo_doc'])
                ->setSerie((string)$d['serie'])
                ->setCorrelativo((string)$d['correlativo'])
                ->setDesMotivoBaja((string)$d['motivo']);
        }
        $voided->setDetails($details);
        return $voided;
    }
}
