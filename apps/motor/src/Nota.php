<?php
declare(strict_types=1);

namespace Facturador\Motor;

use DateTime;
use Greenter\See;
use Greenter\Model\Client\Client;
use Greenter\Model\Company\Address;
use Greenter\Model\Company\Company;
use Greenter\Model\Sale\Note;
use Greenter\Model\Sale\SaleDetail;
use Greenter\Model\Sale\Legend;
use Greenter\Ws\Services\SunatEndpoints;

/**
 * Emisor de notas de crédito (tipo 07) y débito (tipo 08).
 */
final class Nota
{
    /**
     * @param array<string,mixed> $payload
     * @return array<string,mixed>
     */
    public function emitir(array $payload): array
    {
        $modo   = $payload['modo'] ?? 'beta';
        $tenant = $payload['tenant'] ?? [];
        $cpe    = $payload['comprobante'] ?? [];

        $endpoint = $modo === 'prod' ? SunatEndpoints::FE_PRODUCCION : SunatEndpoints::FE_BETA;

        $cert = trim((string)($tenant['cert_pem'] ?? ''));
        $key  = trim((string)($tenant['cert_key_pem'] ?? ''));
        if ($cert === '' || $key === '') {
            return ['estado' => 'error', 'codigo' => 'NO_CERT', 'mensaje' => 'cert_pem y cert_key_pem requeridos'];
        }

        $see = new See();
        $see->setCertificate($cert . "\n" . $key . "\n");
        $see->setService($endpoint);
        $see->setCredentials(
            ((string)$tenant['ruc']) . ((string)($tenant['usuario_sol'] ?? '')),
            (string)($tenant['clave_sol'] ?? '')
        );

        $note = $this->buildNote($tenant, $cpe);

        try {
            $result = $see->send($note);
        } catch (\Throwable $e) {
            return ['estado' => 'error', 'codigo' => 'MOTOR_EXCEPTION', 'mensaje' => $e->getMessage()];
        }

        $xmlFirmado = $see->getFactory()->getLastXml();
        $hash = $this->extraerHash($xmlFirmado ?? '');

        if ($result === null || !$result->isSuccess()) {
            $err = $result?->getError();
            return [
                'estado' => 'rechazado',
                'xml_firmado' => $xmlFirmado ? base64_encode($xmlFirmado) : null,
                'hash_cpe' => $hash,
                'codigo'  => $err?->getCode() ?? 'UNKNOWN',
                'mensaje' => $err?->getMessage() ?? 'SUNAT rechazó',
            ];
        }

        $cdr = $result->getCdrResponse();
        $code = $cdr?->getCode() ?? '';
        $obs = $cdr?->getNotes() ?? [];
        $estado = ($code === '0' && empty($obs)) ? 'aceptado' : 'aceptado_con_obs';

        return [
            'estado' => $estado,
            'xml_firmado' => base64_encode($xmlFirmado ?? ''),
            'cdr_zip'    => base64_encode($result->getCdrZip() ?? ''),
            'hash_cpe'   => $hash,
            'codigo'     => $code,
            'mensaje'    => $cdr?->getDescription() ?? '',
            'observaciones' => $obs,
        ];
    }

    /**
     * @param array<string,mixed> $tenant
     * @param array<string,mixed> $cpe
     */
    private function buildNote(array $tenant, array $cpe): Note
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

        $receptor = (array)$cpe['receptor'];
        $client = (new Client())
            ->setTipoDoc((string)$receptor['tipo_doc'])
            ->setNumDoc((string)$receptor['num_doc'])
            ->setRznSocial((string)$receptor['razon_social'])
            ->setAddress((new Address())->setDireccion((string)($receptor['direccion'] ?? '-')));

        $totales = (array)$cpe['totales'];
        $ref     = (array)($cpe['referencia'] ?? []);

        $numDocAfectado = sprintf('%s-%s', (string)($ref['serie'] ?? ''), (string)($ref['correlativo'] ?? ''));

        $note = (new Note())
            ->setUblVersion('2.1')
            ->setTipoDoc((string)$cpe['tipo'])                       // 07 o 08
            ->setSerie((string)$cpe['serie'])
            ->setCorrelativo((string)$cpe['correlativo'])
            ->setFechaEmision(new DateTime((string)$cpe['fecha_emision']))
            ->setTipoMoneda((string)($cpe['moneda'] ?? 'PEN'))
            ->setTipDocAfectado((string)($ref['tipo_doc'] ?? '01'))
            ->setNumDocfectado($numDocAfectado)
            ->setCodMotivo((string)($cpe['motivo_codigo'] ?? '01'))
            ->setDesMotivo((string)($cpe['motivo_descripcion'] ?? ''))
            ->setCompany($company)
            ->setClient($client)
            ->setMtoOperGravadas((float)($totales['gravado'] ?? 0))
            ->setMtoOperExoneradas((float)($totales['exonerado'] ?? 0))
            ->setMtoOperInafectas((float)($totales['inafecto'] ?? 0))
            ->setMtoIGV((float)($totales['igv'] ?? 0))
            ->setMtoISC((float)($totales['isc'] ?? 0))
            ->setIcbper((float)($totales['icbper'] ?? 0))
            ->setTotalImpuestos((float)($totales['igv'] ?? 0) + (float)($totales['isc'] ?? 0) + (float)($totales['icbper'] ?? 0))
            ->setMtoImpVenta((float)($totales['total'] ?? 0));

        $details = [];
        foreach ((array)$cpe['items'] as $i) {
            $details[] = (new SaleDetail())
                ->setCodProducto((string)($i['codigo'] ?? ''))
                ->setUnidad((string)($i['unidad'] ?? 'NIU'))
                ->setCantidad((float)$i['cantidad'])
                ->setDescripcion((string)$i['descripcion'])
                ->setMtoBaseIgv((float)$i['valor_unitario'] * (float)$i['cantidad'])
                ->setPorcentajeIgv((float)($i['porcentaje_igv'] ?? 18))
                ->setIgv((float)$i['igv'])
                ->setTipAfeIgv((string)($i['afectacion_igv'] ?? '10'))
                ->setTotalImpuestos((float)$i['igv'])
                ->setMtoValorVenta((float)$i['valor_unitario'] * (float)$i['cantidad'])
                ->setMtoValorUnitario((float)$i['valor_unitario'])
                ->setMtoPrecioUnitario((float)$i['precio_unitario']);
        }
        $note->setDetails($details);

        $note->setLegends([
            (new Legend())
                ->setCode('1000')
                ->setValue($this->montoEnLetras((float)($totales['total'] ?? 0), (string)($cpe['moneda'] ?? 'PEN'))),
        ]);

        return $note;
    }

    private function extraerHash(string $xml): string
    {
        if ($xml === '') return '';
        $doc = new \DOMDocument();
        if (!@$doc->loadXML($xml)) return '';
        $nodes = $doc->getElementsByTagNameNS('http://www.w3.org/2000/09/xmldsig#', 'DigestValue');
        return $nodes->length > 0 ? trim($nodes->item(0)->nodeValue ?? '') : '';
    }

    private function montoEnLetras(float $monto, string $moneda): string
    {
        $entero = (int)floor($monto);
        $centavos = (int)round(($monto - $entero) * 100);
        $palabra = $moneda === 'USD' ? 'DOLARES AMERICANOS' : 'SOLES';
        return sprintf('SON %d CON %02d/100 %s', $entero, $centavos, $palabra);
    }
}
