<?php
declare(strict_types=1);

namespace Facturador\Motor;

use DateTime;
use Greenter\See;
use Greenter\Model\Client\Client;
use Greenter\Model\Company\Address;
use Greenter\Model\Company\Company;
use Greenter\Model\Sale\FormaPagos\FormaPagoContado;
use Greenter\Model\Sale\Invoice;
use Greenter\Model\Sale\Legend;
use Greenter\Model\Sale\SaleDetail;
use Greenter\Ws\Services\SunatEndpoints;

/**
 * Emisor de facturas/boletas hacia SUNAT vía Greenter.
 *
 * Stateless. Cada llamada arma un nuevo See(). Sin caché, sin disco.
 */
final class Emisor
{
    /**
     * @param array<string,mixed> $payload Payload con tenant + comprobante (ver CONTRACT.md).
     * @return array<string,mixed>
     */
    public function emitir(array $payload): array
    {
        $modo = $payload['modo'] ?? 'beta';
        $tenant = $payload['tenant'] ?? [];
        $cpe = $payload['comprobante'] ?? [];

        $endpoint = match ($modo) {
            'prod' => SunatEndpoints::FE_PRODUCCION,
            default => SunatEndpoints::FE_BETA,
        };

        $see = new See();
        $see->setCertificate($this->buildCertPem($tenant));
        $see->setService($endpoint);
        $see->setCredentials(
            ($tenant['ruc'] ?? '') . ($tenant['usuario_sol'] ?? ''),
            (string)($tenant['clave_sol'] ?? '')
        );

        $invoice = $this->buildInvoice($tenant, $cpe);

        try {
            $result = $see->send($invoice);
        } catch (\Throwable $e) {
            return [
                'estado' => 'error',
                'codigo' => 'MOTOR_EXCEPTION',
                'mensaje' => $e->getMessage(),
            ];
        }

        $xmlFirmado = $see->getFactory()->getLastXml();
        $hash = $this->extraerHashCpe($xmlFirmado ?? '');

        if ($result === null || !$result->isSuccess()) {
            $err = $result?->getError();
            return [
                'estado' => 'rechazado',
                'xml_firmado' => $xmlFirmado ? base64_encode($xmlFirmado) : null,
                'hash_cpe' => $hash,
                'codigo' => $err?->getCode() ?? 'UNKNOWN',
                'mensaje' => $err?->getMessage() ?? 'SUNAT rechazó el envío sin detalle',
            ];
        }

        $cdr = $result->getCdrResponse();
        $codigo = $cdr?->getCode() ?? '';
        $observaciones = $cdr?->getNotes() ?? [];
        $estado = ($codigo === '0' && empty($observaciones)) ? 'aceptado' : 'aceptado_con_obs';

        return [
            'estado' => $estado,
            'xml_firmado' => base64_encode($xmlFirmado ?? ''),
            'cdr_zip' => base64_encode($result->getCdrZip() ?? ''),
            'hash_cpe' => $hash,
            'codigo' => $codigo,
            'mensaje' => $cdr?->getDescription() ?? '',
            'observaciones' => $observaciones,
        ];
    }

    /**
     * Greenter espera un PEM con cert + private key concatenados.
     *
     * @param array<string,mixed> $tenant
     */
    private function buildCertPem(array $tenant): string
    {
        $cert = (string)($tenant['cert_pem'] ?? '');
        $key = (string)($tenant['cert_key_pem'] ?? '');
        if ($cert === '' || $key === '') {
            throw new \RuntimeException('cert_pem y cert_key_pem son requeridos en tenant');
        }
        return trim($cert) . "\n" . trim($key) . "\n";
    }

    /**
     * @param array<string,mixed> $tenant
     * @param array<string,mixed> $cpe
     */
    private function buildInvoice(array $tenant, array $cpe): Invoice
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
                    ->setUrbanizacion((string)($tenant['urbanizacion'] ?? '-'))
                    ->setDireccion((string)($tenant['direccion_fiscal'] ?? '-'))
                    ->setCodLocal('0000')
            );

        $receptor = (array)$cpe['receptor'];
        $client = (new Client())
            ->setTipoDoc((string)$receptor['tipo_doc'])
            ->setNumDoc((string)$receptor['num_doc'])
            ->setRznSocial((string)$receptor['razon_social'])
            ->setAddress(
                (new Address())->setDireccion((string)($receptor['direccion'] ?? '-'))
            );

        $totales = (array)$cpe['totales'];

        $invoice = (new Invoice())
            ->setUblVersion('2.1')
            ->setTipoOperacion((string)($cpe['tipo_operacion'] ?? '0101'))
            ->setTipoDoc((string)$cpe['tipo'])
            ->setSerie((string)$cpe['serie'])
            ->setCorrelativo((string)$cpe['correlativo'])
            ->setFechaEmision(new DateTime((string)$cpe['fecha_emision']))
            ->setFormaPago(new FormaPagoContado())
            ->setTipoMoneda((string)($cpe['moneda'] ?? 'PEN'))
            ->setCompany($company)
            ->setClient($client)
            ->setMtoOperGravadas((float)($totales['gravado'] ?? 0))
            ->setMtoOperExoneradas((float)($totales['exonerado'] ?? 0))
            ->setMtoOperInafectas((float)($totales['inafecto'] ?? 0))
            ->setMtoOperGratuitas((float)($totales['gratuito'] ?? 0))
            ->setMtoIGV((float)($totales['igv'] ?? 0))
            ->setMtoISC((float)($totales['isc'] ?? 0))
            ->setIcbper((float)($totales['icbper'] ?? 0))
            ->setTotalImpuestos((float)($totales['igv'] ?? 0) + (float)($totales['isc'] ?? 0) + (float)($totales['icbper'] ?? 0))
            ->setValorVenta((float)($totales['gravado'] ?? 0) + (float)($totales['exonerado'] ?? 0) + (float)($totales['inafecto'] ?? 0))
            ->setSubTotal((float)($totales['total'] ?? 0))
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
        $invoice->setDetails($details);

        $invoice->setLegends([
            (new Legend())
                ->setCode('1000')
                ->setValue($this->montoEnLetras((float)($totales['total'] ?? 0), (string)($cpe['moneda'] ?? 'PEN'))),
        ]);

        return $invoice;
    }

    private function extraerHashCpe(string $xmlFirmado): string
    {
        if ($xmlFirmado === '') {
            return '';
        }
        $doc = new \DOMDocument();
        if (!@$doc->loadXML($xmlFirmado)) {
            return '';
        }
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
