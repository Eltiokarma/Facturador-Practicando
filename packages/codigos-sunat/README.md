# Catálogos SUNAT

Tablas de referencia que SUNAT publica y que usamos en el sistema. La fuente oficial está en el [Anexo 8 de la R.S. 097-2012/SUNAT](https://www.sunat.gob.pe/legislacion/superin/2012/anexos/anexoA-097-2012.pdf) y sus actualizaciones.

## Catálogos relevantes (lista no exhaustiva)

| Código | Nombre | Uso |
|--------|--------|-----|
| 01 | Tipos de comprobante de pago | factura, boleta, NC, ND, etc. |
| 02 | Tipos de operación | venta interna, exportación, etc. |
| 06 | Tipos de documento de identidad del adquirente | DNI, RUC, CE, pasaporte |
| 07 | Tipo de afectación del IGV | gravado, exonerado, inafecto, etc. |
| 08 | Tipos de cálculo del ISC | por unidad, ad valórem, etc. |
| 09 | Tipos de nota de crédito | anulación, corrección, devolución, etc. |
| 10 | Tipos de nota de débito | intereses, aumento de valor, etc. |
| 16 | Tipos de moneda | PEN, USD, EUR |
| 17 | Tipos de tributo | IGV (1000), ISC (2000), ICBPER (7152), etc. |
| 51 | Tipo de operación adicional para boletas (sustento) | |
| 53 | Códigos de cargos/descuentos | |
| 59 | Estado de la consulta CDR | |

## Estructura de archivos

- `*.json` — cada catálogo como JSON estructurado. El backend Go los embebe en compile time con `//go:embed`.
- `versiones.md` — registro de cuándo se actualizó cada catálogo.

(Por completar — esto es scaffolding.)
