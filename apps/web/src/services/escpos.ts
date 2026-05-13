// Helpers para construir comandos ESC/POS. Funcionan con cualquier
// impresora térmica que use el estándar EPSON ESC/POS sobre Bluetooth
// SPP, BLE characteristic o USB. Tickets de 80mm con 42 caracteres por
// línea (32 si la impresora es 58mm — configurable).

const ESC = 0x1b;
const GS = 0x1d;
const LF = 0x0a;

export type Alineacion = "left" | "center" | "right";

export class EscPos {
  private chunks: number[] = [];
  private cols: number;

  constructor(cols = 42) {
    this.cols = cols;
  }

  raw(...bytes: number[]): this {
    this.chunks.push(...bytes);
    return this;
  }

  init(): this {
    return this.raw(ESC, 0x40); // ESC @
  }

  text(s: string): this {
    // Convertimos a CP437/CP858 — la mayoría de térmicas usan ese mapeo
    // por default. Para acentos del español funciona ok salvo casos raros.
    for (const ch of s) {
      const code = ch.charCodeAt(0);
      if (code < 0x80) {
        this.chunks.push(code);
      } else {
        // Mapeo mínimo a CP437 (los más comunes en español):
        const map: Record<string, number> = {
          "á": 0xa0, "é": 0x82, "í": 0xa1, "ó": 0xa2, "ú": 0xa3,
          "Á": 0xb5, "É": 0x90, "Í": 0xd6, "Ó": 0xe0, "Ú": 0xe9,
          "ñ": 0xa4, "Ñ": 0xa5,
          "ü": 0x81, "Ü": 0x9a,
          "°": 0xf8, "¿": 0xa8, "¡": 0xad,
          "—": 0x2d, "–": 0x2d, "…": 0x2e,
        };
        this.chunks.push(map[ch] ?? 0x3f); // ? si no mapeable
      }
    }
    return this;
  }

  ln(s = ""): this {
    return this.text(s).raw(LF);
  }

  align(a: Alineacion): this {
    const v = a === "left" ? 0 : a === "center" ? 1 : 2;
    return this.raw(ESC, 0x61, v);
  }

  bold(on: boolean): this {
    return this.raw(ESC, 0x45, on ? 1 : 0);
  }

  size(w: 1 | 2, h: 1 | 2): this {
    // GS ! n : n = (w-1)<<4 | (h-1)
    const n = ((w - 1) & 0x0f) << 4 | ((h - 1) & 0x0f);
    return this.raw(GS, 0x21, n);
  }

  hr(char = "-"): this {
    return this.ln(char.repeat(this.cols));
  }

  // Imprime dos columnas: label pegado a la izquierda, value pegado a la derecha,
  // ambos en la misma línea. Si no entran, value se trunca a la derecha.
  kv(label: string, value: string): this {
    const remaining = this.cols - label.length;
    if (remaining <= 1) {
      // Label demasiado largo, lo separamos en dos líneas
      return this.ln(label).align("right").ln(value).align("left");
    }
    const padded = value.padStart(remaining);
    return this.ln(label + padded);
  }

  // Centra texto en self.cols
  center(s: string): this {
    return this.align("center").ln(s).align("left");
  }

  feed(lines = 3): this {
    for (let i = 0; i < lines; i++) this.chunks.push(LF);
    return this;
  }

  cut(): this {
    // GS V m : m=0 corte completo, m=1 parcial. En térmicas baratas
    // a veces no hay cuchilla — la línea adicional anterior evita
    // quedar pegado al borde.
    return this.raw(GS, 0x56, 0x00);
  }

  qr(data: string, sizeModule = 6): this {
    // QR Code: bloque de comandos GS ( k para impresoras EPSON-compat.
    const bytes = new TextEncoder().encode(data);
    const pL = (bytes.length + 3) & 0xff;
    const pH = ((bytes.length + 3) >> 8) & 0xff;
    return this
      // Modelo 2
      .raw(GS, 0x28, 0x6b, 0x04, 0x00, 0x31, 0x41, 0x32, 0x00)
      // Tamaño del módulo (1-16)
      .raw(GS, 0x28, 0x6b, 0x03, 0x00, 0x31, 0x43, sizeModule)
      // Error correction L
      .raw(GS, 0x28, 0x6b, 0x03, 0x00, 0x31, 0x45, 0x30)
      // Store data
      .raw(GS, 0x28, 0x6b, pL, pH, 0x31, 0x50, 0x30)
      .raw(...bytes)
      // Print
      .raw(GS, 0x28, 0x6b, 0x03, 0x00, 0x31, 0x51, 0x30);
  }

  build(): Uint8Array {
    return new Uint8Array(this.chunks);
  }
}
