#!/usr/bin/env python3
"""Genera docs/manual.pdf — manual del Facturador self-hosted Perú.

Uso:
    pip install reportlab
    python3 docs/manual.py        # crea docs/manual.pdf

No hace falta correrlo si ya tenés docs/manual.pdf en el repo. Está acá
para que puedas regenerarlo si cambia el contenido.
"""
from pathlib import Path

from reportlab.lib import colors
from reportlab.lib.enums import TA_JUSTIFY, TA_CENTER, TA_LEFT
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle, getSampleStyleSheet
from reportlab.lib.units import cm
from reportlab.platypus import (
    BaseDocTemplate,
    Frame,
    PageBreak,
    PageTemplate,
    Paragraph,
    Spacer,
    Table,
    TableStyle,
    KeepTogether,
)


OUT = Path(__file__).parent / "manual.pdf"


# -----------------------------------------------------------------------------
# Estilos
# -----------------------------------------------------------------------------
def build_styles():
    base = getSampleStyleSheet()
    s = {}
    s["title"] = ParagraphStyle(
        "title",
        parent=base["Title"],
        fontName="Helvetica-Bold",
        fontSize=28,
        leading=34,
        spaceAfter=18,
        alignment=TA_CENTER,
        textColor=colors.HexColor("#0f172a"),
    )
    s["subtitle"] = ParagraphStyle(
        "subtitle",
        parent=base["Normal"],
        fontName="Helvetica",
        fontSize=14,
        leading=18,
        spaceAfter=6,
        alignment=TA_CENTER,
        textColor=colors.HexColor("#475569"),
    )
    s["h1"] = ParagraphStyle(
        "h1",
        parent=base["Heading1"],
        fontName="Helvetica-Bold",
        fontSize=20,
        leading=24,
        spaceBefore=18,
        spaceAfter=10,
        textColor=colors.HexColor("#0f172a"),
        keepWithNext=1,
    )
    s["h2"] = ParagraphStyle(
        "h2",
        parent=base["Heading2"],
        fontName="Helvetica-Bold",
        fontSize=14,
        leading=18,
        spaceBefore=12,
        spaceAfter=6,
        textColor=colors.HexColor("#1e293b"),
        keepWithNext=1,
    )
    s["h3"] = ParagraphStyle(
        "h3",
        parent=base["Heading3"],
        fontName="Helvetica-Bold",
        fontSize=11,
        leading=15,
        spaceBefore=8,
        spaceAfter=4,
        textColor=colors.HexColor("#334155"),
        keepWithNext=1,
    )
    s["body"] = ParagraphStyle(
        "body",
        parent=base["BodyText"],
        fontName="Helvetica",
        fontSize=10.5,
        leading=15,
        spaceAfter=6,
        alignment=TA_JUSTIFY,
        textColor=colors.HexColor("#0f172a"),
    )
    s["note"] = ParagraphStyle(
        "note",
        parent=s["body"],
        leftIndent=10,
        rightIndent=10,
        textColor=colors.HexColor("#334155"),
        backColor=colors.HexColor("#fef9c3"),
        borderColor=colors.HexColor("#fde047"),
        borderWidth=0.5,
        borderPadding=8,
        spaceBefore=4,
        spaceAfter=10,
    )
    s["code"] = ParagraphStyle(
        "code",
        parent=base["Code"],
        fontName="Courier",
        fontSize=9,
        leading=12,
        leftIndent=10,
        backColor=colors.HexColor("#f1f5f9"),
        borderColor=colors.HexColor("#cbd5e1"),
        borderWidth=0.5,
        borderPadding=6,
        spaceBefore=4,
        spaceAfter=10,
        textColor=colors.HexColor("#0f172a"),
    )
    s["bullet"] = ParagraphStyle(
        "bullet",
        parent=s["body"],
        leftIndent=18,
        bulletIndent=6,
        spaceAfter=3,
    )
    s["caption"] = ParagraphStyle(
        "caption",
        parent=base["Italic"],
        fontName="Helvetica-Oblique",
        fontSize=9,
        leading=11,
        textColor=colors.HexColor("#64748b"),
        alignment=TA_CENTER,
        spaceAfter=10,
    )
    s["toc_entry"] = ParagraphStyle(
        "toc_entry",
        parent=base["Normal"],
        fontName="Helvetica",
        fontSize=11,
        leading=18,
        textColor=colors.HexColor("#0f172a"),
    )
    return s


# -----------------------------------------------------------------------------
# Page footer
# -----------------------------------------------------------------------------
def draw_footer(canvas, doc):
    canvas.saveState()
    canvas.setFont("Helvetica", 8)
    canvas.setFillColor(colors.HexColor("#94a3b8"))
    canvas.drawString(2 * cm, 1.3 * cm, "Facturador Self-Hosted Perú — Manual")
    canvas.drawRightString(
        A4[0] - 2 * cm, 1.3 * cm, f"página {doc.page}"
    )
    canvas.restoreState()


def make_doc():
    doc = BaseDocTemplate(
        str(OUT),
        pagesize=A4,
        leftMargin=2.2 * cm,
        rightMargin=2.2 * cm,
        topMargin=2.0 * cm,
        bottomMargin=2.0 * cm,
        title="Manual Facturador Self-Hosted Perú",
        author="Proyecto Facturador",
    )
    frame = Frame(
        doc.leftMargin, doc.bottomMargin,
        doc.width, doc.height,
        id="normal",
    )
    doc.addPageTemplates([
        PageTemplate(id="body", frames=frame, onPage=draw_footer),
    ])
    return doc


# -----------------------------------------------------------------------------
# Helpers
# -----------------------------------------------------------------------------
def P(text, style):
    # Permite ** para negrita y _ para itálica en el texto fuente.
    text = text.replace("&", "&amp;")
    return Paragraph(text, style)


def H1(text, S, story, numbered_index=None):
    story.append(PageBreak())
    if numbered_index is not None:
        text = f"{numbered_index}. {text}"
    story.append(Paragraph(text, S["h1"]))


def H2(text, S, story):
    story.append(Paragraph(text, S["h2"]))


def H3(text, S, story):
    story.append(Paragraph(text, S["h3"]))


def body(text, S, story):
    story.append(Paragraph(text, S["body"]))


def note(text, S, story):
    story.append(Paragraph("<b>Nota.</b> " + text, S["note"]))


def code(text, S, story):
    text = text.replace("<", "&lt;").replace(">", "&gt;")
    text = text.replace("\n", "<br/>")
    story.append(Paragraph(text, S["code"]))


def bullets(items, S, story):
    for it in items:
        story.append(Paragraph("• " + it, S["bullet"]))
    story.append(Spacer(1, 4))


def glossary_table(rows, S):
    data = [["Término", "Qué significa, en cristiano"]]
    for term, definition in rows:
        data.append([
            Paragraph(f"<b>{term}</b>", S["body"]),
            Paragraph(definition, S["body"]),
        ])
    tbl = Table(data, colWidths=[3.8 * cm, None], repeatRows=1)
    tbl.setStyle(TableStyle([
        ("BACKGROUND", (0, 0), (-1, 0), colors.HexColor("#0f172a")),
        ("TEXTCOLOR", (0, 0), (-1, 0), colors.white),
        ("FONTNAME", (0, 0), (-1, 0), "Helvetica-Bold"),
        ("FONTSIZE", (0, 0), (-1, 0), 10),
        ("ALIGN", (0, 0), (-1, 0), "LEFT"),
        ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ("BACKGROUND", (0, 1), (-1, -1), colors.HexColor("#f8fafc")),
        ("ROWBACKGROUNDS", (0, 1), (-1, -1), [colors.HexColor("#f8fafc"), colors.white]),
        ("GRID", (0, 0), (-1, -1), 0.3, colors.HexColor("#cbd5e1")),
        ("LEFTPADDING", (0, 0), (-1, -1), 6),
        ("RIGHTPADDING", (0, 0), (-1, -1), 6),
        ("TOPPADDING", (0, 0), (-1, -1), 6),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 6),
    ]))
    return tbl


def codes_table(rows, S):
    data = [["Código", "Significado", "Qué hacer"]]
    for c, m, q in rows:
        data.append([
            Paragraph(f"<b>{c}</b>", S["body"]),
            Paragraph(m, S["body"]),
            Paragraph(q, S["body"]),
        ])
    tbl = Table(data, colWidths=[2.0 * cm, 6.5 * cm, None], repeatRows=1)
    tbl.setStyle(TableStyle([
        ("BACKGROUND", (0, 0), (-1, 0), colors.HexColor("#0f172a")),
        ("TEXTCOLOR", (0, 0), (-1, 0), colors.white),
        ("FONTNAME", (0, 0), (-1, 0), "Helvetica-Bold"),
        ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ("ROWBACKGROUNDS", (0, 1), (-1, -1), [colors.HexColor("#f8fafc"), colors.white]),
        ("GRID", (0, 0), (-1, -1), 0.3, colors.HexColor("#cbd5e1")),
        ("LEFTPADDING", (0, 0), (-1, -1), 6),
        ("RIGHTPADDING", (0, 0), (-1, -1), 6),
        ("TOPPADDING", (0, 0), (-1, -1), 6),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 6),
    ]))
    return tbl


# -----------------------------------------------------------------------------
# Contenido
# -----------------------------------------------------------------------------
def build_story(S):
    story = []

    # ------------------------------ PORTADA --------------------------------
    story.append(Spacer(1, 5 * cm))
    story.append(P("Facturador Self-Hosted Perú", S["title"]))
    story.append(P("Manual del proyecto, teoría y glosario", S["subtitle"]))
    story.append(Spacer(1, 1 * cm))
    story.append(P(
        "Un manual para entender de qué se trata este software,<br/>"
        "qué hace cada pieza, por qué fue construido así<br/>"
        "y cómo usarlo aunque no sepas programar.",
        S["subtitle"],
    ))
    story.append(Spacer(1, 4 * cm))
    story.append(P("Versión inicial — 2026", S["caption"]))

    # ------------------------------ ÍNDICE --------------------------------
    story.append(PageBreak())
    story.append(P("Índice", S["h1"]))
    toc = [
        ("1.  Para qué sirve este software", 4),
        ("2.  Un poco de contexto: la facturación electrónica en Perú", 5),
        ("3.  Por qué self-hosted y no un servicio en la nube", 7),
        ("4.  Las piezas del sistema", 8),
        ("5.  El viaje de una factura, paso a paso", 11),
        ("6.  Conceptos técnicos explicados sin jerga", 13),
        ("7.  Cómo se configura todo", 17),
        ("8.  Tu primera factura beta", 19),
        ("9.  Cuando SUNAT rechaza: cómo leer el error", 21),
        ("10. Pasar a producción", 22),
        ("11. Mantenimiento, respaldos y certificado", 23),
        ("12. Roadmap: qué falta y cuándo", 24),
        ("13. Glosario de términos", 25),
        ("14. Anexo A — Códigos de error SUNAT más comunes", 30),
        ("15. Anexo B — Mapa de archivos del proyecto", 32),
    ]
    for label, page in toc:
        dots = "." * max(3, 70 - len(label) - len(str(page)))
        story.append(P(f"{label}  {dots}  pág. {page}", S["toc_entry"]))

    # ---------------------------- 1. PARA QUÉ ------------------------------
    H1("Para qué sirve este software", S, story, 1)
    body(
        "Este proyecto resuelve un problema concreto: <b>en Perú, todo "
        "negocio formal está obligado a emitir comprobantes electrónicos "
        "(facturas, boletas, notas de crédito y notas de débito) y "
        "enviarlos a SUNAT</b>. Hacer eso a mano es imposible y las "
        "empresas comerciales que ofrecen ese servicio cobran "
        "mensualidades — entre S/ 30 y S/ 150 al mes — que se acumulan y "
        "que muchas veces no se justifican técnicamente.",
        S, story,
    )
    body(
        "Este software hace exactamente lo mismo que esos servicios "
        "comerciales (Nubefact, Efact, Facpro, Kesito y compañía), pero "
        "<b>corre en tu propia computadora o en un servidor que vos "
        "alquilás</b>. Vos sos el dueño de tus datos, de tu certificado, "
        "y de cuándo y cómo se emiten tus comprobantes. No pagás "
        "mensualidad porque no estás contratando un servicio, sino "
        "corriendo un programa.",
        S, story,
    )
    body(
        "La analogía más simple: usar un facturador comercial es como "
        "alquilarle a alguien una caja registradora y pagarle todos los "
        "meses por usarla. Este proyecto es como comprarte la caja "
        "registradora una sola vez — bueno, gratis en este caso — y "
        "ponerla en tu negocio.",
        S, story,
    )
    note(
        "Este software <b>no es</b> un PSE ni un OSE — no firmamos "
        "comprobantes en nombre de terceros ni custodiamos certificados "
        "de nadie. Es un programa que vos corrés, con tu propio "
        "certificado, en tu propia infraestructura.",
        S, story,
    )

    H2("Qué emite, qué no emite", S, story)
    body("Hoy emite:", S, story)
    bullets([
        "<b>Factura</b> (tipo 01): comprobante para ventas entre empresas con RUC.",
        "<b>Boleta de venta</b> (tipo 03): comprobante para personas naturales con DNI.",
        "<b>Nota de crédito</b> (tipo 07): corrección o devolución de una factura/boleta anterior.",
        "<b>Nota de débito</b> (tipo 08): aumento del monto a cobrar respecto a un comprobante anterior.",
    ], S, story)
    body(
        "Hoy <b>no</b> emite:",
        S, story,
    )
    bullets([
        "<b>Guía de Remisión Electrónica (GRE)</b>: queda para una versión posterior. SUNAT la exige obligatoriamente desde julio de 2026, así que está en el roadmap.",
        "<b>Comprobantes de retención y percepción</b>: aplicables si sos agente de retención o percepción.",
        "<b>Resúmenes diarios y comunicaciones de baja</b>: para anular boletas o anular comprobantes ya enviados.",
    ], S, story)

    H2("Para quién es", S, story)
    body(
        "Pensado para <b>micro, pequeñas y medianas empresas (MYPE)</b> "
        "que quieran soberanía sobre su facturación y no quieran pagar "
        "una suscripción. Concretamente:",
        S, story,
    )
    bullets([
        "Un restaurante, ferretería, bodega, taller, consultorio, "
        "estudio contable — cualquier negocio que emita comprobantes con frecuencia.",
        "Un contador que quiera correr un servidor propio para emitir comprobantes de varios clientes (multi-RUC).",
        "Un dueño con dos o tres empresas que quiera centralizar la emisión.",
    ], S, story)
    body(
        "<b>No es para</b> grandes contribuyentes (PRICOS) que facturen "
        "más de 300 UIT al año (S/ 1.650.000 al 2026): la ley les exige "
        "usar un OSE registrado, no auto-emisión.",
        S, story,
    )

    # ---------------------------- 2. CONTEXTO ------------------------------
    H1("Un poco de contexto: la facturación electrónica en Perú", S, story, 2)
    body(
        "Antes de meternos en cómo funciona el software, conviene entender "
        "qué es lo que SUNAT exige. Esto sirve para que después, cuando "
        "leas las partes técnicas, todo tenga sentido.",
        S, story,
    )

    H2("Qué es un Comprobante de Pago Electrónico (CPE)", S, story)
    body(
        "En Perú, hace más de una década, SUNAT migró toda la "
        "facturación a formato electrónico. Un CPE es básicamente un "
        "<b>archivo XML</b> (un tipo de archivo de texto estructurado) "
        "que contiene los datos del comprobante: emisor, receptor, "
        "fecha, productos, montos, impuestos. Ese XML <b>tiene que estar "
        "firmado digitalmente</b> con un certificado del contribuyente, "
        "y <b>tiene que enviarse a SUNAT</b> en un formato técnico "
        "específico llamado SOAP.",
        S, story,
    )
    body(
        "SUNAT responde con otro archivo, llamado <b>CDR</b> "
        "(Constancia de Recepción): un ZIP que adentro tiene un XML que "
        "dice si tu comprobante fue aceptado, aceptado con observaciones, "
        "o rechazado. Ese CDR es la prueba legal de que el comprobante "
        "fue presentado.",
        S, story,
    )
    note(
        "Los tres archivos que importan para cada comprobante son: "
        "(1) el XML que enviaste, (2) el CDR que SUNAT te devolvió, y "
        "(3) la representación impresa (PDF) que le entregás al cliente. "
        "Los tres tienen que conservarse <b>mínimo 5 años</b>.",
        S, story,
    )

    H2("Los actores: emisor, OSE y PSE", S, story)
    body(
        "SUNAT diseñó el sistema con tres roles posibles:",
        S, story,
    )
    bullets([
        "<b>Emisor electrónico</b>: el contribuyente que emite. Vos. "
        "El que vende algo y debe entregar comprobante.",
        "<b>OSE (Operador de Servicios Electrónicos)</b>: empresa "
        "autorizada por SUNAT que <b>valida</b> los comprobantes antes "
        "de que lleguen a SUNAT. Obligatorio para PRICOS grandes. Para "
        "MYPE y medianas no aplica.",
        "<b>PSE (Proveedor de Servicios Electrónicos)</b>: empresa que "
        "ofrece la <b>infraestructura</b> de emisión como servicio. "
        "Acá entran los Nubefact, Efact, etc. Cobran mensualidad.",
    ], S, story)
    body(
        "Este software te permite operar como <b>emisor electrónico "
        "directo</b>, sin pasar por un PSE. SUNAT llama a esta "
        "modalidad <b>SEE del Contribuyente</b> (Sistema de Emisión "
        "Electrónica del Contribuyente).",
        S, story,
    )

    H2("Qué necesitás conseguir antes de usar el software", S, story)
    body(
        "Hay tres cosas que solo vos podés conseguir, no las puede "
        "automatizar ningún software. Son trámites en SUNAT.",
        S, story,
    )

    H3("1. Tu RUC activo y habido", S, story)
    body(
        "Si ya estás operando formalmente, ya lo tenés. Tu RUC debe "
        "estar en estado <b>activo</b> (no suspendido) y <b>habido</b> "
        "(SUNAT te puede ubicar en tu domicilio fiscal). Si no, no podés "
        "facturar electrónicamente ni con este software ni con ningún otro.",
        S, story,
    )

    H3("2. Tu Certificado Digital Tributario (CDT)", S, story)
    body(
        "Es un <b>archivo .p12</b> que SUNAT te entrega gratis. Sirve "
        "para firmar digitalmente cada CPE que emitís. Lo bajás desde "
        "Clave SOL, en la sección de Comprobantes de Pago. Tiene "
        "vigencia 3 años. La passphrase del archivo la elegís vos al "
        "generarlo — anotala bien, sin ella el archivo no sirve. Mirá "
        "el documento <i>docs/obtener-cdt.md</i> en el repo para el "
        "paso a paso.",
        S, story,
    )

    H3("3. Un usuario secundario Clave SOL", S, story)
    body(
        "SUNAT no quiere que uses tu Clave SOL principal (la que usás "
        "vos para entrar al portal) en un sistema automatizado. Te "
        "exige crear un <b>usuario secundario</b> con un permiso muy "
        "específico: <i>Emisión electrónica de comprobantes desde los "
        "sistemas del contribuyente</i>. Sin este usuario, SUNAT "
        "responde error 0111 a todo intento de envío. El procedimiento "
        "completo está en <i>docs/crear-usuario-secundario.md</i>.",
        S, story,
    )
    note(
        "<b>Regla de oro</b>: la Clave SOL principal NUNCA va al "
        "software. Si la ves en un archivo de configuración, eso es un "
        "bug de seguridad. Solo el usuario secundario, con permisos "
        "limitados, va al sistema.",
        S, story,
    )

    H2("Plazos legales que importan", S, story)
    bullets([
        "<b>3 días calendario</b> para enviar el CPE a SUNAT, contados "
        "desde el día siguiente a la fecha de emisión. Si pasa el plazo, "
        "SUNAT rechaza aunque el comprobante ya esté en manos del cliente.",
        "<b>5 años</b> mínimo de conservación de cada CPE y CDR.",
        "<b>UIT 2026: S/ 5.500</b>. La sanción por no emitir cuando "
        "debías es del 50% UIT (S/ 2.750) por comprobante, con rebaja "
        "del 90% si subsanás voluntariamente.",
    ], S, story)

    # ----------------------- 3. POR QUÉ SELF-HOSTED -----------------------
    H1("Por qué self-hosted y no un servicio en la nube", S, story, 3)
    body(
        "La pregunta natural es: <b>si Nubefact, Kesito, Efact y "
        "compañía ya existen y funcionan, ¿para qué construir esto?</b> "
        "La respuesta tiene tres partes.",
        S, story,
    )

    H2("Costo a lo largo del tiempo", S, story)
    body(
        "Un facturador comercial cobra entre S/ 30 y S/ 150 mensuales. "
        "Eso son entre S/ 360 y S/ 1.800 al año, todos los años, sin "
        "parar. Para un negocio chico que emite pocas facturas es una "
        "carga significativa. Para un negocio que ya creció, es un "
        "gasto recurrente que no aporta más valor a medida que crece.",
        S, story,
    )
    body(
        "Un VPS donde correr este software cuesta entre US$ 4 y US$ 10 "
        "al mes (S/ 15 a S/ 38). Una laptop vieja conectada a "
        "internet puede correrlo gratis si tu volumen es bajo.",
        S, story,
    )

    H2("Soberanía sobre tus datos", S, story)
    body(
        "Con un facturador comercial, <b>tus comprobantes, tu certificado "
        "y tus datos de clientes están en servidores de un tercero</b>. "
        "Eso significa que si la empresa quiebra, sube precios, te corta "
        "el servicio por una disputa, o tiene una brecha de seguridad, "
        "vos sufrís. Tu información se va con ellos.",
        S, story,
    )
    body(
        "Con self-hosting, tu certificado .p12 nunca sale de tu "
        "servidor. La passphrase que lo desbloquea solo existe en la "
        "memoria del proceso, nunca se guarda en disco. Los XMLs y CDRs "
        "los respaldás vos donde quieras. Si en algún momento querés "
        "cambiar de software, te llevás todo.",
        S, story,
    )

    H2("Transparencia técnica", S, story)
    body(
        "El código es abierto. Vos podés leer (o pedirle a un técnico "
        "de confianza que lea) qué hace cada parte. Con un servicio "
        "comercial, tenés que confiar a ciegas en que están haciendo "
        "lo que dicen, cobrándote lo justo, y cumpliendo SUNAT bien.",
        S, story,
    )

    H2("La contra honesta", S, story)
    body(
        "Self-hosting no es gratis en esfuerzo. Hay que:",
        S, story,
    )
    bullets([
        "Instalar y configurar el software una vez.",
        "Mantener al día el servidor o la laptop donde corre.",
        "Hacer respaldos regulares de los CPE y CDR (responsabilidad legal).",
        "Renovar el certificado cada 3 años (gratis pero hay que acordarse).",
    ], S, story)
    body(
        "Si nada de esto te interesa o no tenés cómo manejarlo, un "
        "facturador comercial sigue siendo una opción legítima. Este "
        "software es para quien valora la soberanía y el ahorro a "
        "largo plazo por sobre la comodidad de pagar mensualidad.",
        S, story,
    )

    # ---------------------------- 4. PIEZAS ------------------------------
    H1("Las piezas del sistema", S, story, 4)
    body(
        "Un sistema de facturación no es una sola pieza, son varias. "
        "Pensalo como una cocina industrial: hay quien toma el pedido, "
        "quien cocina, quien empaca y quien entrega. En este software "
        "pasa lo mismo. Veamos cada pieza con esa analogía.",
        S, story,
    )

    H2("La cara visible: el frontend web", S, story)
    body(
        "Es la <b>página que ves en el navegador</b> cuando entrás al "
        "sistema. Te muestra formularios para crear facturas, listados "
        "de comprobantes emitidos, configuración. Está construida en "
        "<b>React</b>, una tecnología muy difundida para hacer "
        "interfaces web. Es una PWA (Progressive Web App), lo que "
        "significa que <b>funciona también sin internet</b>: si se "
        "te cae la conexión, podés seguir armando facturas y el sistema "
        "las enviará a SUNAT cuando la conexión vuelva.",
        S, story,
    )
    body(
        "<i>Analogía:</i> el mesero del restaurante. Toma tu pedido y "
        "te muestra qué hay. No cocina nada, solo es el intermediario "
        "entre vos y la cocina.",
        S, story,
    )

    H2("El que decide: el backend API", S, story)
    body(
        "Es el cerebro del sistema. Recibe los pedidos del frontend, "
        "<b>valida</b> que los datos sean correctos (por ejemplo, que "
        "un RUC tenga 11 dígitos, que las cantidades sean positivas), "
        "<b>recalcula los totales</b> y los impuestos (no se confía en "
        "lo que viene del navegador), genera el correlativo siguiente "
        "de la serie, y le pasa el trabajo al motor de facturación. "
        "Está escrito en <b>Go</b>, un lenguaje de programación "
        "moderno conocido por ser rápido y confiable.",
        S, story,
    )
    body(
        "<i>Analogía:</i> el jefe de cocina. Recibe la comanda, "
        "verifica que tenga sentido, organiza los ingredientes y le "
        "pasa la receta al cocinero especialista.",
        S, story,
    )

    H2("El especialista: el motor de facturación", S, story)
    body(
        "Es la pieza que <b>habla el idioma de SUNAT</b>. Toma los "
        "datos del comprobante, los serializa en un XML con el formato "
        "exacto que SUNAT exige (un estándar llamado UBL 2.1), lo "
        "firma digitalmente con tu certificado, lo envuelve en un "
        "sobre técnico (SOAP), lo envía al servidor de SUNAT, y "
        "espera la respuesta (el CDR). Está escrito en <b>PHP</b> "
        "porque usa una librería peruana muy probada que se llama "
        "<b>Greenter</b>, que ya resuelve todos los detalles de la "
        "regulación SUNAT.",
        S, story,
    )
    body(
        "<i>Analogía:</i> el chef especializado en cocina molecular. "
        "Sabe exactamente cómo combinar ácido cítrico con esferas de "
        "calcio y montar el plato como exige el manual. El jefe de "
        "cocina no necesita saber esos detalles, solo le pasa el pedido.",
        S, story,
    )

    H2("La memoria: la base de datos", S, story)
    body(
        "Es donde se <b>guardan permanentemente</b> los datos: tenants "
        "(empresas configuradas), usuarios del sistema, series y "
        "correlativos, comprobantes emitidos con su estado, bitácora "
        "de envíos a SUNAT. Es <b>PostgreSQL</b>, una base de datos "
        "robusta usada en miles de proyectos empresariales.",
        S, story,
    )
    body(
        "<i>Analogía:</i> el archivero del restaurante. Guarda todos "
        "los recibos, comandas, listas de proveedores, ventas del mes.",
        S, story,
    )
    note(
        "En la versión actual (MVP), la base de datos todavía no está "
        "totalmente activada — los comprobantes se guardan en archivos "
        "dentro de la carpeta <i>./data/</i>. La migración a PostgreSQL "
        "completa es uno de los próximos pasos.",
        S, story,
    )

    H2("La cola: Redis y el worker", S, story)
    body(
        "SUNAT a veces está lento, a veces se cae, y nunca responde "
        "instantáneamente. Si cada vez que el cajero presiona <i>Emitir</i> "
        "tuviéramos que esperar a SUNAT antes de mostrarle algo, la "
        "experiencia sería pésima. La solución es una <b>cola</b>: "
        "cuando se emite, el comprobante se guarda y se mete en una "
        "lista de pendientes, y un proceso separado (el <i>worker</i>) "
        "los va sacando de a uno y enviando a SUNAT en segundo plano.",
        S, story,
    )
    body(
        "<b>Redis</b> es el software que mantiene esa cola en memoria.",
        S, story,
    )
    body(
        "<i>Analogía:</i> el sistema de tickets impresos en la cocina. "
        "Cuando entra un pedido, se cuelga en la línea; el cocinero "
        "los va tomando en orden mientras el mesero ya volvió a atender "
        "otra mesa.",
        S, story,
    )
    note(
        "En el MVP actual la emisión es <b>sincrónica</b> (el sistema "
        "espera a SUNAT antes de responder al cliente). La cola se "
        "activa en el próximo milestone. La pieza Redis ya está "
        "levantada en Docker, lista para cuando se conecte.",
        S, story,
    )

    H2("El orquestador: Docker Compose", S, story)
    body(
        "Todas estas piezas (frontend, backend, motor, base de datos, "
        "Redis) son programas separados que tienen que arrancar en "
        "orden y comunicarse entre sí. <b>Docker</b> es una tecnología "
        "que las empaqueta cada una en su propio contenedor aislado. "
        "<b>Docker Compose</b> es la herramienta que coordina todos "
        "los contenedores con un solo comando.",
        S, story,
    )
    body(
        "Por eso al final, todo lo que tenés que hacer para arrancar "
        "el sistema es ejecutar:",
        S, story,
    )
    code("docker compose -f docker-compose.yml -f docker-compose.beta.yml up -d", S, story)
    body(
        "Y todo se levanta solo.",
        S, story,
    )

    # ----------------------- 5. VIAJE DE UNA FACTURA ----------------------
    H1("El viaje de una factura, paso a paso", S, story, 5)
    body(
        "Pongámonos en el ejemplo concreto. Vos atendés a un cliente y "
        "decidís emitir una factura. Esto es lo que pasa adentro del "
        "sistema, paso por paso. Te lo cuento como si fueras un "
        "auditor mirando desde afuera.",
        S, story,
    )

    H3("Paso 1 — Vos llenás el formulario", S, story)
    body(
        "Abrís el frontend en el navegador, elegís el cliente o lo "
        "creás (RUC, razón social), agregás los ítems con cantidad y "
        "precio, ponés método de pago. El navegador te muestra el "
        "total calculado con IGV en tiempo real.",
        S, story,
    )

    H3("Paso 2 — El frontend manda los datos al backend", S, story)
    body(
        "Cuando hacés click en <b>Emitir</b>, el frontend serializa "
        "todos esos datos como un JSON (otro formato de texto "
        "estructurado, más legible que XML) y lo envía al backend Go "
        "como una petición HTTP POST a la ruta <i>/api/v1/facturas</i>.",
        S, story,
    )

    H3("Paso 3 — El backend valida y recalcula", S, story)
    body(
        "El backend Go revisa: ¿el RUC del receptor tiene 11 dígitos? "
        "¿la fecha de emisión es válida? ¿los ítems tienen cantidad y "
        "precio positivos? ¿la serie corresponde al tipo de comprobante? "
        "Después, <b>recalcula todos los totales y el IGV desde cero</b>. "
        "El motivo es de seguridad y correctitud: si el frontend tuviera "
        "un bug o si alguien con malas intenciones modificara los "
        "valores enviados, no queremos terminar emitiendo un comprobante "
        "con un IGV calculado mal.",
        S, story,
    )

    H3("Paso 4 — Se asigna el correlativo", S, story)
    body(
        "Cada serie (F001, F002, B001, etc.) tiene su propio contador. "
        "El backend toma el siguiente número de manera <b>atómica</b>: "
        "esto quiere decir que si llegaran dos emisiones al mismo "
        "tiempo, el sistema garantiza que no asigne el mismo número a "
        "ambas. Esto es crítico, porque SUNAT rechaza comprobantes "
        "duplicados.",
        S, story,
    )

    H3("Paso 5 — Se envía el trabajo al motor", S, story)
    body(
        "El backend Go arma un JSON con toda la info y se lo manda al "
        "motor PHP por HTTP, dentro de la red privada de Docker. "
        "Junto con los datos del comprobante, le pasa el certificado "
        "ya descifrado en memoria y las credenciales del usuario "
        "secundario Clave SOL.",
        S, story,
    )

    H3("Paso 6 — El motor genera el XML, lo firma y lo envía", S, story)
    body(
        "El motor PHP usa Greenter para construir el XML UBL 2.1 con "
        "el formato exacto que SUNAT exige. Después aplica la <b>firma "
        "digital XAdES-BES</b> con tu certificado — esto es lo que "
        "le da validez legal al documento. Después lo envuelve en un "
        "sobre SOAP con WS-Security (donde van las credenciales SOL), "
        "y lo manda al servidor de SUNAT (beta si estás probando, "
        "producción si ya estás operando en serio).",
        S, story,
    )

    H3("Paso 7 — SUNAT responde con el CDR", S, story)
    body(
        "Después de unos segundos (a veces 1, a veces 10, a veces más), "
        "SUNAT responde. La respuesta puede ser:",
        S, story,
    )
    bullets([
        "<b>Aceptado</b> (código 0, sin observaciones): todo perfecto.",
        "<b>Aceptado con observaciones</b> (código 0 + warnings): el "
        "comprobante es válido pero hay algo a corregir para próximas emisiones.",
        "<b>Rechazado</b> (código > 0): el comprobante no fue aceptado. "
        "Hay que corregir y reemitir.",
    ], S, story)

    H3("Paso 8 — Se guardan los artefactos", S, story)
    body(
        "El sistema guarda en disco:",
        S, story,
    )
    bullets([
        "El XML firmado que se envió a SUNAT.",
        "El CDR (ZIP) que SUNAT devolvió.",
        "Un archivo JSON con el resumen y estado.",
    ], S, story)
    body(
        "Estos tres archivos son tu prueba legal. SUNAT los puede "
        "pedir en una fiscalización. Hay que conservarlos 5 años.",
        S, story,
    )

    H3("Paso 9 — Se genera el PDF para el cliente", S, story)
    body(
        "Con los mismos datos, el sistema arma una representación "
        "impresa bonita (A4 y/o ticket de 80mm) con tu logo, datos, y "
        "un código QR que contiene el hash del CPE. Eso es lo que le "
        "entregás al cliente — por mail, por WhatsApp, o impreso.",
        S, story,
    )
    note(
        "El PDF bonito está en el roadmap inmediato. El MVP actual "
        "todavía no lo genera, pero el XML y el CDR ya quedan guardados.",
        S, story,
    )

    # ----------------------- 6. CONCEPTOS TÉCNICOS ------------------------
    H1("Conceptos técnicos explicados sin jerga", S, story, 6)
    body(
        "Esta sección desarma los términos técnicos más importantes "
        "para que cuando aparezcan en mensajes de error, documentación, "
        "o conversaciones con técnicos, sepas exactamente de qué se "
        "está hablando.",
        S, story,
    )

    H2("XML — el idioma de SUNAT", S, story)
    body(
        "XML es un formato de archivo de texto que estructura "
        "información con etiquetas, parecido al HTML de las páginas "
        "web. Un fragmento simplificado:",
        S, story,
    )
    code(
        "<cac:AccountingSupplierParty>\n"
        "    <cbc:CustomerAssignedAccountID>20123456789</cbc:CustomerAssignedAccountID>\n"
        "    <cbc:AdditionalAccountID>6</cbc:AdditionalAccountID>\n"
        "</cac:AccountingSupplierParty>",
        S, story,
    )
    body(
        "Lo importante: SUNAT exige que el XML tenga una estructura "
        "exacta. Cada etiqueta tiene que estar en su lugar. Cada "
        "campo tiene que cumplir reglas (longitudes, códigos válidos, "
        "etc.). Por eso usamos Greenter — porque construir este XML "
        "a mano es muy propenso a errores.",
        S, story,
    )

    H2("UBL 2.1 — el estándar internacional", S, story)
    body(
        "UBL son las siglas de <i>Universal Business Language</i>. Es "
        "un estándar internacional para formatos de documentos "
        "comerciales (facturas, órdenes de compra, notas de envío). "
        "SUNAT lo adoptó y le agregó las particularidades peruanas "
        "(catálogos de impuestos, tipos de documento de identidad, "
        "códigos de afectación de IGV).",
        S, story,
    )
    body(
        "Cuando alguien dice <i>UBL 2.1</i>, está hablando de la versión "
        "del estándar que SUNAT exige. Otras ediciones (UBL 2.0, UBL "
        "2.2) existen pero no aplican.",
        S, story,
    )

    H2("Firma digital y XAdES-BES", S, story)
    body(
        "Cuando vos firmás un papel con tu mano, ese trazo es difícil "
        "de imitar y prueba que vos validaste el documento. La firma "
        "digital es el equivalente para archivos: un proceso "
        "matemático que toma el contenido del archivo y tu certificado "
        "(que es como tu identidad digital), y produce una <b>huella "
        "única</b> que se adjunta al archivo. Cualquiera puede "
        "verificar después que esa huella corresponde a tu certificado "
        "y que el archivo no fue modificado.",
        S, story,
    )
    body(
        "<b>XAdES-BES</b> (XML Advanced Electronic Signature - Basic "
        "Electronic Signature) es la variante específica de firma "
        "digital para XML que SUNAT exige. Define exactamente qué "
        "partes del XML se firman y cómo se incluye la firma adentro "
        "del mismo archivo.",
        S, story,
    )

    H2("Certificado .p12 y passphrase", S, story)
    body(
        "Tu certificado digital es un archivo con extensión <i>.p12</i> "
        "(también llamado PKCS#12). Adentro tiene dos cosas: <b>tu "
        "llave privada</b> (un número enorme que solo vos conocés) y "
        "<b>tu certificado</b> (que enlaza esa llave con tu identidad "
        "fiscal). El archivo está encriptado con una passphrase que "
        "vos elegiste cuando lo generaste en SUNAT.",
        S, story,
    )
    body(
        "Para firmar un CPE el sistema necesita las dos cosas: leer "
        "el archivo y desbloquearlo con la passphrase. Por eso, este "
        "software:",
        S, story,
    )
    bullets([
        "Pide la passphrase al arrancar (no la guarda en ningún archivo).",
        "Descifra el .p12 una vez y mantiene las claves en memoria.",
        "Si el servidor se reinicia, hay que volver a introducir la passphrase.",
    ], S, story)
    body(
        "Esto es una decisión consciente de diseño: queremos que <b>en "
        "ningún momento la passphrase esté escrita en disco</b>. Si "
        "alguien roba el servidor apagado, tiene el .p12 pero no la "
        "passphrase, y por lo tanto no puede firmar comprobantes.",
        S, story,
    )

    H2("SOAP — el camión que lleva el XML a SUNAT", S, story)
    body(
        "SOAP es un protocolo viejo (de los años 2000) que sigue "
        "siendo usado por sistemas gubernamentales y empresariales. "
        "Es como un sobre: adentro va tu XML firmado, y por fuera "
        "tiene un encabezado con datos de quién lo envía y cómo se "
        "autentica. SUNAT recibe ese sobre en una dirección "
        "específica de internet (un endpoint).",
        S, story,
    )

    H2("WS-Security: cómo se autentica el envío", S, story)
    body(
        "WS-Security es la parte del sobre SOAP donde van las "
        "credenciales. Específicamente, SUNAT usa el modo "
        "<i>UsernameToken con PasswordText</i>: tu usuario "
        "(<i>RUC + usuario secundario</i>) y la contraseña en texto "
        "plano. ¿Texto plano no es inseguro? Sí, pero todo viaja por "
        "HTTPS (conexión cifrada), así que en la práctica está "
        "protegido. Es decisión de SUNAT, no nuestra.",
        S, story,
    )

    H2("CDR — la respuesta de SUNAT", S, story)
    body(
        "CDR significa <b>Constancia de Recepción</b>. Es la respuesta "
        "que SUNAT te devuelve después de procesar el comprobante. "
        "Llega como un archivo ZIP que adentro tiene un XML "
        "(<i>ApplicationResponse</i>) con el estado:",
        S, story,
    )
    bullets([
        "Código de respuesta (0 = aceptado, otros = rechazo o aviso).",
        "Descripción legible: <i>'La Factura número F001-1 ha sido aceptada'</i>.",
        "Observaciones (si las hay).",
    ], S, story)
    body(
        "El CDR es la <b>prueba legal</b> de que SUNAT recibió el "
        "comprobante. Sin CDR, ante una fiscalización, no podés "
        "probar que enviaste.",
        S, story,
    )

    H2("Beta y producción: los dos mundos de SUNAT", S, story)
    body(
        "SUNAT mantiene dos ambientes idénticos:",
        S, story,
    )
    bullets([
        "<b>Beta (homologación)</b>: pensado para probar. Los comprobantes "
        "que envíes acá <b>no tienen efecto legal</b>. SUNAT los recibe, "
        "valida y responde igual que en producción, pero no quedan "
        "registrados como reales. Sirve para verificar que tu sistema "
        "funciona antes de emitir en serio.",
        "<b>Producción</b>: el real. Lo que enviés acá <b>sí tiene "
        "valor legal</b>. Hay que estar 100% seguro antes de tocarlo.",
    ], S, story)
    body(
        "Este software permite cambiar entre ambos con una sola "
        "variable de configuración: <i>SUNAT_MODE=beta</i> o "
        "<i>SUNAT_MODE=prod</i>. Lo demás se mantiene igual.",
        S, story,
    )

    H2("IGV y la regla del 18%", S, story)
    body(
        "El Impuesto General a las Ventas (IGV) en Perú es del 18% "
        "sobre el valor de venta. Pero no todos los ítems están "
        "afectos: hay productos exonerados (frutas, vegetales sin "
        "procesar), inafectos (servicios financieros), gratuitos, "
        "etc. SUNAT tiene una tabla (el catálogo 07) con los códigos "
        "de afectación. El sistema usa estos códigos para cada ítem y "
        "calcula el IGV correspondiente.",
        S, story,
    )

    H2("Multi-tenant: una instalación, varias empresas", S, story)
    body(
        "Tenant significa <i>inquilino</i>: cada empresa (cada RUC) "
        "que usa el sistema es un tenant. Multi-tenant quiere decir "
        "que una sola instalación del software puede manejar varias "
        "empresas a la vez, manteniendo los datos de cada una "
        "completamente separados.",
        S, story,
    )
    body(
        "Por ejemplo, si sos contador y tenés cinco clientes, podés "
        "correr una sola instalación de este software y cargar los "
        "cinco RUCs con sus respectivos certificados. Cada cliente "
        "ve solo lo suyo.",
        S, story,
    )

    H2("MVP — Minimum Viable Product", S, story)
    body(
        "MVP es un término del desarrollo de software. Significa "
        "<i>la versión más simple posible que ya hace algo útil</i>. "
        "La idea: en vez de construir todo de una y tardar un año, "
        "construís primero el flujo crítico (emitir UNA factura "
        "contra SUNAT beta) y a partir de ahí vas agregando "
        "funcionalidad. Lo que tenés ahora es el MVP.",
        S, story,
    )

    # ---------------------------- 7. CÓMO SE CONFIGURA --------------------
    H1("Cómo se configura todo", S, story, 7)
    body(
        "Esta sección no es tutorial paso a paso de instalación — eso "
        "está en <i>docs/instalacion.md</i>. Acá explicamos <b>qué "
        "configura cada cosa y por qué importa</b>.",
        S, story,
    )

    H2("El archivo .env: el centro de control", S, story)
    body(
        "Todo se configura en un archivo llamado <i>.env</i> que vos "
        "armás copiando <i>.env.example</i>. Adentro hay variables "
        "como esta:",
        S, story,
    )
    code(
        "TENANT_RUC=20123456789\n"
        "TENANT_RAZON_SOCIAL=MI EMPRESA SAC\n"
        "CERT_PASSPHRASE=loquesea123\n"
        "SUNAT_MODE=beta",
        S, story,
    )
    body(
        "Cada variable controla algo específico. Las más importantes:",
        S, story,
    )

    H3("Identidad de tu empresa (TENANT_*)", S, story)
    bullets([
        "<b>TENANT_RUC</b>: tu RUC de 11 dígitos.",
        "<b>TENANT_RAZON_SOCIAL</b>: nombre legal de la empresa.",
        "<b>TENANT_NOMBRE_COMERCIAL</b>: nombre comercial (puede ser el mismo).",
        "<b>TENANT_DIRECCION_FISCAL</b>: dirección registrada en SUNAT.",
        "<b>TENANT_UBIGEO</b>: código de 6 dígitos del distrito según SUNAT.",
        "<b>TENANT_DEPARTAMENTO, TENANT_PROVINCIA, TENANT_DISTRITO</b>: en mayúsculas.",
    ], S, story)
    body(
        "Estos datos deben coincidir <b>exactamente</b> con lo que "
        "SUNAT tiene registrado. Si la razón social difiere por una "
        "letra, SUNAT rechaza.",
        S, story,
    )

    H3("Credenciales de envío (TENANT_USUARIO_SOL, TENANT_CLAVE_SOL)", S, story)
    body(
        "Estas son las credenciales del <b>usuario secundario</b> que "
        "creaste en Clave SOL. Para pruebas en beta podés usar el "
        "clásico <i>MODDATOS / MODDATOS</i>; para producción, sí o sí "
        "el usuario tuyo.",
        S, story,
    )

    H3("El certificado (CERT_PATH, CERT_PASSPHRASE)", S, story)
    body(
        "El archivo .p12 va físicamente en la carpeta <i>certs/</i> "
        "del proyecto. La variable <i>CERT_PATH</i> dice dónde "
        "encontrarlo dentro del contenedor Docker (por defecto "
        "<i>/app/certs/cert.p12</i>). La variable <i>CERT_PASSPHRASE</i> "
        "es la contraseña que vos elegiste al generar el .p12.",
        S, story,
    )
    note(
        "El archivo .env no se sube al repositorio de código (está "
        "excluido). Si lo subís por error, la passphrase queda "
        "expuesta y tenés que regenerar el certificado en SUNAT. "
        "Cuidalo como cuidás tu DNI.",
        S, story,
    )

    H3("Modo SUNAT (SUNAT_MODE)", S, story)
    body(
        "Tiene dos valores posibles: <i>beta</i> o <i>prod</i>. Empezá "
        "siempre en beta. Solo cuando ya validaste que el flujo "
        "completo funciona, cambialo a prod.",
        S, story,
    )

    H3("Secretos del servidor (API_JWT_SECRET, POSTGRES_PASSWORD)", S, story)
    body(
        "Son contraseñas internas del sistema que no se las das a "
        "nadie. Tienen que ser largas y aleatorias. Para generarlas "
        "podés correr <i>openssl rand -hex 32</i> en una terminal.",
        S, story,
    )

    # ----------------------- 8. PRIMERA FACTURA BETA ----------------------
    H1("Tu primera factura beta", S, story, 8)
    body(
        "Asumiendo que ya:",
        S, story,
    )
    bullets([
        "Tenés tu .p12 en <i>certs/cert.p12</i>.",
        "Llenaste el <i>.env</i> con tus datos.",
        "Instalaste Docker.",
    ], S, story)
    body(
        "Estos son los comandos que vas a correr, en orden, en una "
        "terminal dentro de la carpeta del proyecto:",
        S, story,
    )

    H3("1. Levantar todo el stack", S, story)
    code("docker compose -f docker-compose.yml -f docker-compose.beta.yml up -d", S, story)
    body(
        "Docker descarga las imágenes, las arma, las arranca. La "
        "primera vez puede tardar varios minutos. Las siguientes son "
        "casi instantáneas.",
        S, story,
    )

    H3("2. Verificar que esté todo arriba", S, story)
    code("curl http://localhost:8080/health", S, story)
    body(
        "Si responde algo como <i>{\"status\":\"ok\",\"sunat_mode\":\"beta\"}</i>, "
        "el backend está funcionando.",
        S, story,
    )
    code("curl http://localhost:8080/ready", S, story)
    body(
        "Si responde <i>{\"motor\":\"ok\"}</i>, el motor PHP también "
        "está vivo y conectado.",
        S, story,
    )

    H3("3. Emitir la primera factura demo", S, story)
    code("./scripts/probar-factura.sh", S, story)
    body(
        "Este script envía una factura con datos de prueba: serie "
        "<i>F001</i>, correlativo <i>1</i>, un solo ítem por S/ 100 "
        "más IGV, total S/ 118.",
        S, story,
    )

    H3("4. Leer la respuesta", S, story)
    body(
        "El script imprime la respuesta de SUNAT en pantalla y te "
        "dice dónde quedaron los archivos. Si todo salió bien, vas a "
        "ver algo así:",
        S, story,
    )
    code(
        "🎉 SUNAT aceptó tu primera factura beta.\n"
        "   Mirá ./data/01-F001-00000001/",
        S, story,
    )
    body(
        "Adentro de esa carpeta, los tres archivos clave:",
        S, story,
    )
    bullets([
        "<b>registro.json</b>: resumen del comprobante.",
        "<b>firmado.xml</b>: el UBL firmado que se envió a SUNAT.",
        "<b>cdr.zip</b>: la Constancia de Recepción.",
    ], S, story)
    body(
        "Estos son los tres archivos que tenés que conservar 5 años "
        "según la ley.",
        S, story,
    )

    H3("5. Emitir otra (cambiando correlativo)", S, story)
    code("CORRELATIVO=2 ./scripts/probar-factura.sh", S, story)
    body(
        "O cambiando la serie:",
        S, story,
    )
    code("SERIE=F002 CORRELATIVO=1 ./scripts/probar-factura.sh", S, story)

    # ----------------------- 9. CUANDO SUNAT RECHAZA ----------------------
    H1("Cuando SUNAT rechaza: cómo leer el error", S, story, 9)
    body(
        "SUNAT rechaza con un código numérico y un mensaje. La regla "
        "general:",
        S, story,
    )
    bullets([
        "<b>Códigos 1000-1999</b>: errores de estructura del XML "
        "(suelen ser bugs del software, no de tus datos).",
        "<b>Códigos 2000-2999</b>: errores de validación de datos "
        "(RUC inválido, fecha mal, totales no cuadran).",
        "<b>Códigos 3000-3999</b>: errores de validación SUNAT más "
        "estrictos (afectación de IGV mal asignada, montos negativos).",
        "<b>Códigos 4000+</b>: <b>warnings</b>, no rechazo. El "
        "comprobante igual queda aceptado pero hay algo a mejorar.",
    ], S, story)
    body(
        "Cuando SUNAT rechaza, el campo <i>mensaje</i> casi siempre "
        "te dice exactamente qué está mal. El Anexo A al final de "
        "este manual lista los códigos más frecuentes con qué hacer "
        "ante cada uno.",
        S, story,
    )

    H2("Si el motor o la API no responden", S, story)
    body(
        "A veces el problema no es SUNAT, sino algo local. Comandos "
        "útiles para diagnosticar:",
        S, story,
    )
    code("docker compose ps", S, story)
    body("Muestra qué contenedores están arriba y cuáles no.", S, story)
    code("docker compose logs api motor", S, story)
    body(
        "Muestra los logs de los servicios. Si algo falló, el motivo "
        "casi siempre aparece ahí.",
        S, story,
    )
    code("docker compose restart api motor", S, story)
    body(
        "Reinicia los servicios. Después de reiniciar, hay que "
        "volver a poner la <i>CERT_PASSPHRASE</i> en el <i>.env</i> "
        "(es a propósito).",
        S, story,
    )

    # ----------------------- 10. PASAR A PRODUCCIÓN -----------------------
    H1("Pasar a producción", S, story, 10)
    body(
        "Solo después de validar el flujo en beta. Idealmente, "
        "después de:",
        S, story,
    )
    bullets([
        "Emitir varias facturas y boletas beta sin errores.",
        "Verificar que los datos del emisor coinciden con SUNAT.",
        "Tener creado el usuario secundario real (no MODDATOS).",
        "Tener un plan de respaldos.",
    ], S, story)
    body(
        "Los pasos para el cambio:",
        S, story,
    )
    bullets([
        "Apagar el stack: <i>docker compose down</i>.",
        "En <i>.env</i>: cambiar <i>SUNAT_MODE=beta</i> a <i>SUNAT_MODE=prod</i>.",
        "En <i>.env</i>: cambiar <i>TENANT_USUARIO_SOL</i> y <i>TENANT_CLAVE_SOL</i> "
        "por los del usuario secundario real (no MODDATOS).",
        "Levantar sin el override beta: <i>docker compose up -d</i>.",
        "Probar con una factura a un cliente conocido antes de "
        "abrirle el sistema al equipo.",
    ], S, story)
    note(
        "Después de cualquier reinicio del contenedor, el sistema "
        "vuelve a leer el .env y por lo tanto la passphrase del "
        "certificado. Esto es a propósito: queremos que la "
        "passphrase esté en un archivo que vos controlás, no "
        "guardada permanentemente.",
        S, story,
    )

    # ----------------------- 11. MANTENIMIENTO -----------------------
    H1("Mantenimiento, respaldos y certificado", S, story, 11)

    H2("Qué respaldar", S, story)
    bullets([
        "<b>Carpeta <i>data/</i></b>: contiene todos los XML firmados "
        "y CDR. Es lo más importante. Sin esto, ante una fiscalización, "
        "no podés probar lo que emitiste.",
        "<b>Carpeta <i>certs/</i></b>: contiene tu .p12. Si lo "
        "perdés, lo regenerás desde Clave SOL (gratis).",
        "<b>Archivo <i>.env</i></b>: contiene tu configuración. Sin "
        "esto reconfigurás todo a mano.",
        "<b>Base de datos PostgreSQL</b>: cuando esté activa, hay que "
        "hacer dumps regulares con <i>pg_dump</i>.",
    ], S, story)

    H2("Con qué frecuencia respaldar", S, story)
    body(
        "Recomendación realista para un negocio chico:",
        S, story,
    )
    bullets([
        "<b>Diario</b>: copia automática de <i>data/</i> a un disco "
        "externo o a un servicio en la nube cifrado.",
        "<b>Semanal</b>: copia completa de todo, idealmente fuera del "
        "servidor donde corre el sistema.",
    ], S, story)

    H2("Cuándo renovar el certificado", S, story)
    body(
        "El CDT vale 3 años. El sistema te debería avisar 30 días "
        "antes (esa funcionalidad está en el roadmap). El "
        "procedimiento es el mismo que cuando lo bajaste por primera "
        "vez (Clave SOL → Empresas → CDT → Generar), con una nueva "
        "passphrase. Reemplazás el .p12 en <i>certs/cert.p12</i> y "
        "actualizás <i>CERT_PASSPHRASE</i> en el <i>.env</i>.",
        S, story,
    )

    H2("Actualizar el software", S, story)
    body(
        "El software va a tener releases. La idea es que actualizar "
        "sea tan simple como:",
        S, story,
    )
    code("git pull\ndocker compose pull\ndocker compose up -d", S, story)
    body(
        "Antes de actualizar producción, hacé respaldo de "
        "<i>data/</i>. Las versiones intentarán ser compatibles "
        "hacia atrás, pero respaldar siempre es la regla.",
        S, story,
    )

    # ----------------------- 12. ROADMAP -----------------------
    H1("Roadmap: qué falta y cuándo", S, story, 12)
    body(
        "El estado actual es un MVP funcional. Lo que sigue, en "
        "orden aproximado de prioridad:",
        S, story,
    )

    H2("Próximo (semanas)", S, story)
    bullets([
        "Cola asíncrona Redis + worker (eliminar la espera sincrónica a SUNAT).",
        "Persistencia completa en PostgreSQL (correlativos atómicos, bitácora de envíos).",
        "Notas de crédito y débito.",
        "Boletas con resumen diario.",
        "PDF bonito con QR.",
        "Autenticación de usuarios JWT (multi-usuario por tenant).",
    ], S, story)

    H2("Mediano plazo (meses)", S, story)
    bullets([
        "Interfaz web completa (hoy solo es scaffolding).",
        "Funcionamiento offline real con cola local en el navegador.",
        "Importación masiva desde Excel.",
        "Notificaciones por email al cliente.",
        "Reportes y exportes contables.",
        "Multi-tenant real con onboarding de nuevos RUCs desde la UI.",
    ], S, story)

    H2("Antes de julio 2026", S, story)
    bullets([
        "Guía de Remisión Electrónica (GRE), que SUNAT exigirá obligatoriamente.",
    ], S, story)

    H2("Futuro lejano", S, story)
    bullets([
        "Distribución como binarios nativos (sin Docker).",
        "Empaquetado para Windows/Mac/Linux con instalador gráfico.",
        "Posible app móvil para emisión rápida.",
    ], S, story)

    # ----------------------- 13. GLOSARIO -----------------------
    H1("Glosario de términos", S, story, 13)
    body(
        "Términos técnicos, regulatorios y de software que aparecen "
        "en el proyecto, en la documentación SUNAT, o cuando hables "
        "con un técnico. Ordenados alfabéticamente.",
        S, story,
    )

    glossary_entries = [
        ("API", "Conjunto de rutas web que un programa expone para que otros programas le hablen. En este proyecto, la API (escrita en Go) es lo que el frontend usa para emitir comprobantes."),
        ("Backend", "La parte del software que corre en el servidor, fuera del navegador. Decide qué guardar, qué validar, qué responder. En este proyecto, el backend está escrito en Go."),
        ("Beta (SUNAT)", "Ambiente de pruebas de SUNAT. Lo que enviás acá no tiene efecto legal. Sirve para validar tu sistema antes de pasar a producción."),
        ("Boleta", "Comprobante para personas naturales (consumidores finales). Identificado con tipo 03. Serie empieza con B."),
        ("CDR", "Constancia de Recepción. Archivo ZIP que SUNAT te devuelve después de que recibe tu comprobante. Adentro está el veredicto: aceptado, aceptado con observaciones, o rechazado. Es prueba legal."),
        ("CDT", "Certificado Digital Tributario. Archivo .p12 que SUNAT entrega gratis. Sirve para firmar digitalmente los CPE. Vigencia 3 años."),
        ("Clave SOL", "Las credenciales (usuario y contraseña) que SUNAT te da para acceder a sus servicios online. Tenés una principal (la del titular del RUC) y podés crear secundarias con permisos limitados."),
        ("Codigo de afectación IGV", "Código del catálogo SUNAT 07 que define cómo afecta el IGV a un ítem: 10 gravado, 20 exonerado, 30 inafecto, etc."),
        ("Comprobante de Pago Electrónico (CPE)", "Cualquiera de los documentos electrónicos que SUNAT obliga a emitir: factura, boleta, nota de crédito, nota de débito, guía de remisión."),
        ("Container (Docker)", "Una caja aislada donde corre un programa. Tiene su propio sistema de archivos, su propia red interna, sus propias dependencias. Si se rompe, no afecta al resto."),
        ("Correlativo", "Número secuencial dentro de una serie. F001-1, F001-2, F001-3... SUNAT exige que sean consecutivos sin saltos."),
        ("Docker", "Tecnología para empaquetar programas con todas sus dependencias y correrlos de manera aislada en cualquier servidor."),
        ("Docker Compose", "Herramienta de Docker para describir varios contenedores que se coordinan entre sí en un solo archivo YAML."),
        ("Emisor electrónico", "El contribuyente que emite comprobantes. Vos."),
        ("Endpoint", "Una dirección de internet a la que un programa le pide algo. SUNAT tiene endpoints distintos para beta y producción."),
        ("env (archivo .env)", "Archivo de configuración con variables como TENANT_RUC, CERT_PASSPHRASE, etc. Cada programa que lo necesita las lee."),
        ("Factura", "Comprobante para operaciones entre empresas o con consumidores con RUC. Tipo 01. Serie empieza con F."),
        ("Firma digital", "Proceso matemático que prueba que un archivo fue validado por el dueño de un certificado y que el contenido no fue alterado."),
        ("Frontend", "La parte del software que ves en el navegador. En este proyecto, hecha con React."),
        ("Git", "Sistema para versionar código fuente. Permite ver el historial de cambios, volver atrás, trabajar en ramas paralelas."),
        ("Go", "Lenguaje de programación creado por Google. Compilado, rápido, popular para servidores. Usado en el backend de este proyecto."),
        ("Greenter", "Librería peruana de código abierto que resuelve la generación de UBL, firma XAdES, envío SOAP a SUNAT y parseo de CDR. Es el corazón del motor de facturación."),
        ("GRE", "Guía de Remisión Electrónica. Documento que acompaña el traslado físico de mercadería. Obligatoria desde julio 2026."),
        ("Hash", "Una huella única calculada matemáticamente sobre un archivo. Si cambia un solo carácter, el hash es completamente diferente. SUNAT incluye el hash del CPE en el código QR."),
        ("HTTP / HTTPS", "El protocolo que usan los navegadores y APIs para comunicarse. HTTPS es la versión cifrada."),
        ("ICBPER", "Impuesto al Consumo de Bolsas Plásticas. S/ 0.50 por bolsa (al 2026, sujeto a cambio)."),
        ("IGV", "Impuesto General a las Ventas. 18% sobre el valor de venta de bienes y servicios gravados."),
        ("ISC", "Impuesto Selectivo al Consumo. Aplica a productos específicos (cigarrillos, alcohol, combustibles)."),
        ("JSON", "Formato de texto para intercambiar datos. Más simple y legible que XML. Usado entre el frontend y el backend."),
        ("JWT", "JSON Web Token. Tipo de credencial que el backend emite cuando un usuario se autentica. Sirve para que las siguientes peticiones se identifiquen sin pedir contraseña otra vez."),
        ("Migración (DB)", "Archivo SQL que describe cómo crear o modificar tablas de la base de datos. Se aplican en orden cuando se actualiza el sistema."),
        ("MODDATOS", "Usuario clásico de pruebas SUNAT para el ambiente beta. Sirve para que cualquiera pueda probar el flujo sin haber creado su usuario secundario aún."),
        ("Monorepo", "Estructura de proyecto donde varios componentes (backend, frontend, motor) viven en un solo repositorio Git."),
        ("Motor de facturación", "En este proyecto, el microservicio PHP que genera UBL, firma y envía a SUNAT. Aislado del resto por seguridad."),
        ("MVP", "Minimum Viable Product. La versión más simple que ya hace algo útil."),
        ("Nota de crédito", "Documento que reduce o corrige una factura/boleta previa (descuentos, devoluciones, anulaciones). Tipo 07."),
        ("Nota de débito", "Documento que aumenta el monto de una factura/boleta previa (intereses, aumento de valor). Tipo 08."),
        ("OSE", "Operador de Servicios Electrónicos. Empresa autorizada por SUNAT que valida comprobantes antes de SUNAT. Obligatorio para PRICOS grandes."),
        ("PEM", "Formato de texto para representar certificados y llaves criptográficas. Empieza con <i>-----BEGIN CERTIFICATE-----</i> o similar."),
        ("PHP", "Lenguaje de programación usado en el motor de facturación de este proyecto. Lo usamos porque Greenter, la librería peruana de facturación, está escrita en PHP."),
        ("PKCS#12", "Estándar para empaquetar certificados con su llave privada en un solo archivo cifrado. Extensión .p12 o .pfx."),
        ("PostgreSQL", "Base de datos relacional de código abierto, robusta. Usada en miles de proyectos empresariales. Es la DB de este proyecto."),
        ("PRICO", "Principal Contribuyente. Empresas grandes designadas por SUNAT, con exigencias regulatorias más estrictas (OSE obligatorio)."),
        ("Producción (SUNAT)", "Ambiente real de SUNAT. Lo que enviás tiene valor legal pleno."),
        ("PSE", "Proveedor de Servicios Electrónicos. Empresa que ofrece la infraestructura de emisión como servicio. Nubefact, Efact, etc., son PSE."),
        ("PWA", "Progressive Web App. Página web que también funciona instalada como app, con soporte offline."),
        ("Queue (cola)", "Lista de tareas pendientes que se procesan una por una en segundo plano."),
        ("React", "Librería de JavaScript para construir interfaces de usuario en el navegador. Usada en el frontend."),
        ("Redis", "Base de datos en memoria, muy rápida. Usada para mantener la cola de jobs."),
        ("Reverse proxy", "Programa que recibe pedidos de internet y los reparte a los servicios internos. Caddy y Nginx son ejemplos comunes. Recomendado si exponés el sistema a internet."),
        ("RUC", "Registro Único de Contribuyente. Tu identificación tributaria en Perú. 11 dígitos."),
        ("Self-hosted", "Software que vos corrés en tu propia infraestructura, en oposición a SaaS donde lo corre un tercero."),
        ("SEE", "Sistema de Emisión Electrónica. El conjunto de modalidades por las que se emiten CPE. SEE del Contribuyente es la que usamos acá."),
        ("Serie", "Identificador de 4 caracteres que agrupa un rango de correlativos. F001, F002 para facturas; B001, B002 para boletas."),
        ("SOAP", "Protocolo de comunicación basado en XML. SUNAT lo usa para recibir los CPE."),
        ("SQL", "Lenguaje para consultar y modificar bases de datos relacionales."),
        ("SSL/TLS", "Las tecnologías que cifran el tráfico entre tu computadora y un servidor. Es lo que hace HTTPS seguro."),
        ("Stack", "El conjunto de tecnologías que componen un proyecto. El stack de este proyecto es: Go + React + PHP + PostgreSQL + Redis + Docker."),
        ("Stateless", "Que no guarda estado entre llamadas. El motor PHP de este proyecto es stateless: no tiene DB, si se reinicia no perdió nada."),
        ("SUNAT", "Superintendencia Nacional de Aduanas y de Administración Tributaria. El ente recaudador de Perú."),
        ("Tenant", "En multi-tenant, cada empresa (RUC) que usa el sistema. Una instalación, varios tenants."),
        ("UBL", "Universal Business Language. Estándar internacional para documentos comerciales en XML. SUNAT usa UBL 2.1."),
        ("Ubigeo", "Código de 6 dígitos que SUNAT usa para identificar el departamento, provincia y distrito en Perú."),
        ("UIT", "Unidad Impositiva Tributaria. Valor de referencia que se actualiza cada año. Al 2026: S/ 5.500. Se usa para calcular sanciones y umbrales."),
        ("Usuario secundario Clave SOL", "Usuario adicional con permisos limitados creado bajo tu Clave SOL principal. El que se le da al software para que emita en tu nombre."),
        ("Volumen (Docker)", "Carpeta del servidor host que se monta dentro de un contenedor. Permite que los datos sobrevivan reinicios del contenedor."),
        ("VPS", "Virtual Private Server. Servidor virtual alquilado por mes. Una opción común para correr este software (DigitalOcean, Linode, Hetzner, etc.)."),
        ("Worker", "Proceso que corre en segundo plano sacando tareas de una cola y ejecutándolas. En este proyecto, el worker es el que envía a SUNAT."),
        ("WS-Security", "Estándar para incluir credenciales y firmas en mensajes SOAP. SUNAT lo usa para autenticar el envío."),
        ("XAdES-BES", "Variante de firma digital específica para XML que SUNAT exige. BES significa Basic Electronic Signature."),
        ("XML", "Formato de texto estructurado con etiquetas. El idioma en el que viaja la información a SUNAT."),
        ("YAML", "Formato de texto para configuración, más legible que XML o JSON. Docker Compose usa YAML."),
    ]
    story.append(glossary_table(glossary_entries, S))

    # ----------------------- 14. ANEXO A: CÓDIGOS SUNAT --------------------
    H1("Anexo A — Códigos de error SUNAT más comunes", S, story, 14)
    body(
        "Cuando SUNAT rechaza, devuelve un código numérico y un "
        "mensaje. Esta tabla cubre los más frecuentes:",
        S, story,
    )
    codes_rows = [
        ("0111", "El usuario SOL no tiene perfil para enviar comprobantes electrónicos.", "Crear o asignar permiso 'Emisión electrónica de comprobantes desde los sistemas del contribuyente' al usuario secundario."),
        ("0150", "Las credenciales SOL son incorrectas.", "Verificar TENANT_USUARIO_SOL y TENANT_CLAVE_SOL en .env. Recordar que el usuario va sin el RUC adelante en este software (el sistema lo concatena solo)."),
        ("1032", "El número de documento de identidad no es válido para el tipo indicado.", "Revisar el RUC (11 dígitos), DNI (8), CE, etc., y que el tipo de documento (catálogo 06) corresponda."),
        ("1078", "La factura debe identificar al cliente con RUC.", "Cambiar a boleta o registrar el RUC del cliente."),
        ("2017", "El número de RUC del receptor no existe.", "Verificar el RUC. Puede haber un dígito mal."),
        ("2018", "El RUC del receptor no está activo o no está habido.", "El cliente debe regularizar su situación en SUNAT antes de poder recibir factura."),
        ("2335", "El RUC del emisor no está activo.", "Tu RUC propio está suspendido o de baja. Regularizar primero en SUNAT."),
        ("2800", "Comprobante ya fue presentado anteriormente.", "Ya enviaste este correlativo. No volver a enviar — buscar el CDR original."),
        ("3105", "Los montos no cuadran con los ítems.", "Generalmente un bug de cálculo. Reportar al desarrollador. El software recalcula server-side; este error implica un caso no contemplado."),
        ("3206", "La fecha de emisión es posterior a la fecha actual.", "Verificar la fecha del servidor (zona horaria) y la fecha enviada."),
        ("3211", "Se excedió el plazo de envío (3 días calendario).", "Comprobantes vencidos no pueden enviarse. Anular y emitir uno nuevo con fecha actual."),
        ("4000-4999", "Warnings (no rechazos).", "El comprobante quedó aceptado pero hay observaciones. Revisar para próximas emisiones."),
        ("Connection timeout", "SUNAT no responde a tiempo.", "Reintentar luego. La cola asíncrona maneja esto automáticamente cuando esté activada."),
    ]
    story.append(codes_table(codes_rows, S))

    # ----------------------- 15. ANEXO B: ARCHIVOS -----------------------
    H1("Anexo B — Mapa de archivos del proyecto", S, story, 15)
    body(
        "Si abrís el repositorio, esto es lo que encontrás y qué hace "
        "cada cosa:",
        S, story,
    )
    code(
        "facturador/\n"
        "├── CLAUDE.md                  Contexto del proyecto (para IA y humanos)\n"
        "├── README.md                  Resumen y filosofía\n"
        "├── docker-compose.yml         Define los servicios del stack\n"
        "├── docker-compose.beta.yml    Override para forzar modo beta\n"
        "├── .env.example               Plantilla de configuración\n"
        "│\n"
        "├── apps/\n"
        "│   ├── api/                   Backend Go\n"
        "│   │   ├── cmd/api/main.go    Punto de entrada del programa\n"
        "│   │   ├── internal/\n"
        "│   │   │   ├── cert/          Carga del .p12, descifrado en memoria\n"
        "│   │   │   ├── config/        Lee variables de entorno\n"
        "│   │   │   ├── facturacion/   Modelo de factura y recálculo de totales\n"
        "│   │   │   ├── http/          Rutas y handlers HTTP\n"
        "│   │   │   ├── motor/         Cliente que habla con el motor PHP\n"
        "│   │   │   ├── storage/       Persistencia en filesystem (por ahora)\n"
        "│   │   │   ├── sunat/         Endpoints SUNAT por modo\n"
        "│   │   │   ├── tenant/        Configuración del tenant\n"
        "│   │   │   └── db/migrations/ Scripts SQL para PostgreSQL\n"
        "│   │   └── Dockerfile         Cómo construir la imagen Docker\n"
        "│   │\n"
        "│   ├── motor/                 Motor PHP/Greenter\n"
        "│   │   ├── src/Emisor.php     Lógica de emisión\n"
        "│   │   ├── public/index.php   Servidor HTTP del motor\n"
        "│   │   ├── composer.json      Dependencias PHP\n"
        "│   │   ├── CONTRACT.md        Contrato entre Go y motor PHP\n"
        "│   │   └── Dockerfile\n"
        "│   │\n"
        "│   └── web/                   Frontend React PWA\n"
        "│       ├── src/App.tsx        Componente principal\n"
        "│       ├── package.json       Dependencias JavaScript\n"
        "│       ├── vite.config.ts     Configuración del bundler\n"
        "│       └── Dockerfile\n"
        "│\n"
        "├── packages/\n"
        "│   ├── codigos-sunat/         Catálogos SUNAT (en construcción)\n"
        "│   └── ubl-schemas/           Esquemas XSD UBL 2.1\n"
        "│\n"
        "├── docs/\n"
        "│   ├── adr/0001-stack-motor.md   Por qué Go + PHP\n"
        "│   ├── instalacion.md            Guía paso a paso\n"
        "│   ├── obtener-cdt.md            Cómo bajar el .p12 de SUNAT\n"
        "│   ├── crear-usuario-secundario.md  Cómo crear el usuario SOL secundario\n"
        "│   ├── manual.py                 Script que genera este PDF\n"
        "│   └── manual.pdf                Este documento\n"
        "│\n"
        "├── scripts/\n"
        "│   ├── probar-factura.sh      Emite 1 factura beta de prueba\n"
        "│   └── seed-beta.sh           (Por implementar) crea tenant de prueba\n"
        "│\n"
        "├── certs/                     Tus certificados .p12 (no commiteado)\n"
        "└── data/                      XML firmados y CDR (no commiteado)",
        S, story,
    )

    body(
        "Cualquier archivo que termina en <i>.go</i> es código del "
        "backend Go. Cualquier <i>.php</i> es del motor. <i>.tsx</i> y "
        "<i>.ts</i> son del frontend. <i>.sql</i> son migraciones de "
        "base de datos. <i>.yml</i> son configuraciones de Docker "
        "Compose. <i>.md</i> son documentos como este.",
        S, story,
    )

    # ----------------------- CIERRE -----------------------
    H1("Cierre", S, story)
    body(
        "Si llegaste hasta acá, ya tenés una idea general bastante "
        "completa de qué es este proyecto, cómo funciona y cómo se "
        "usa. No hace falta que entiendas el código línea por línea; "
        "alcanza con tener un mapa mental de las piezas y cómo "
        "conversan entre sí.",
        S, story,
    )
    body(
        "El objetivo del proyecto es muy concreto: <b>que un "
        "contribuyente peruano pueda emitir comprobantes electrónicos "
        "sin pagar mensualidad y sin perder control sobre sus datos</b>. "
        "Todo lo demás — código en Go, motor PHP, Docker, React — son "
        "medios para ese fin.",
        S, story,
    )
    body(
        "Cuando algo no te quede claro, o cuando aparezca un mensaje "
        "de SUNAT que no entiendas, el glosario y el Anexo A "
        "deberían cubrir la mayoría de los casos. Y si no, siempre "
        "podés consultar.",
        S, story,
    )
    body("— Fin del manual —", S, story)

    return story


def main():
    S = build_styles()
    doc = make_doc()
    doc.build(build_story(S))
    print(f"PDF generado: {OUT}")


if __name__ == "__main__":
    main()
