#!/usr/bin/env python3
"""Genera íconos del Facturador en varios tamaños.

Producto: un cuadrado con el color brand (#4f46e5 índigo) y un símbolo de
documento + check sobre fondo claro. Salida:
    apps/web/public/icons/icon-192.png
    apps/web/public/icons/icon-512.png
    apps/web/public/icons/icon-maskable-512.png
    apps/web/public/favicon.png

Uso:
    pip install Pillow
    python3 scripts/build-icons.py
"""
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parent.parent
OUT_ICONS = ROOT / "apps/web/public/icons"
OUT_PUB = ROOT / "apps/web/public"


def build_square(size: int, padding_ratio: float = 0.0) -> Image.Image:
    """Construye el ícono cuadrado. padding_ratio>0 deja espacio extra
    para íconos 'maskable' (Android los recorta en círculo)."""
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)

    # Fondo redondeado con color brand
    pad = int(size * padding_ratio)
    inner = size - 2 * pad
    radius = int(inner * 0.18)
    d.rounded_rectangle(
        (pad, pad, pad + inner, pad + inner),
        radius=radius,
        fill=(79, 70, 229, 255),  # brand-600
    )

    # Hoja de papel (rectángulo blanco con doblez en esquina)
    cx, cy = size / 2, size / 2
    paper_w = inner * 0.46
    paper_h = inner * 0.56
    fold = inner * 0.12
    paper_l = cx - paper_w / 2
    paper_t = cy - paper_h / 2 - inner * 0.02
    paper_r = paper_l + paper_w
    paper_b = paper_t + paper_h
    # esquina con doblez
    d.polygon([
        (paper_l, paper_t),
        (paper_r - fold, paper_t),
        (paper_r, paper_t + fold),
        (paper_r, paper_b),
        (paper_l, paper_b),
    ], fill=(255, 255, 255, 255))
    # doblez (triangulito gris)
    d.polygon([
        (paper_r - fold, paper_t),
        (paper_r - fold, paper_t + fold),
        (paper_r, paper_t + fold),
    ], fill=(224, 231, 255, 255))  # brand-100

    # Líneas de texto en la hoja
    line_h = inner * 0.05
    line_gap = inner * 0.07
    for i in range(3):
        ly = paper_t + inner * 0.18 + i * line_gap
        d.rounded_rectangle(
            (paper_l + inner * 0.05, ly,
             paper_r - inner * 0.05 - (inner * 0.08 if i == 2 else 0), ly + line_h),
            radius=line_h / 2,
            fill=(99, 102, 241, 255),  # brand-500
        )

    # Check verde redondo abajo a la derecha
    check_size = inner * 0.32
    check_cx = paper_r + inner * 0.02
    check_cy = paper_b - inner * 0.02
    d.ellipse(
        (check_cx - check_size / 2, check_cy - check_size / 2,
         check_cx + check_size / 2, check_cy + check_size / 2),
        fill=(16, 185, 129, 255),  # emerald-500
    )
    # Marca de check (líneas blancas)
    cw = check_size * 0.6
    cs = max(2, int(check_size * 0.12))
    d.line(
        [
            (check_cx - cw * 0.32, check_cy + cw * 0.02),
            (check_cx - cw * 0.05, check_cy + cw * 0.28),
            (check_cx + cw * 0.36, check_cy - cw * 0.22),
        ],
        fill=(255, 255, 255, 255),
        width=cs,
        joint="curve",
    )

    return img


def main():
    OUT_ICONS.mkdir(parents=True, exist_ok=True)
    OUT_PUB.mkdir(parents=True, exist_ok=True)

    sizes = {
        OUT_ICONS / "icon-192.png": (192, 0.0),
        OUT_ICONS / "icon-512.png": (512, 0.0),
        OUT_ICONS / "icon-maskable-512.png": (512, 0.12),
        OUT_PUB / "favicon.png": (64, 0.0),
        OUT_PUB / "apple-touch-icon.png": (180, 0.0),
    }

    for out_path, (size, pad_ratio) in sizes.items():
        img = build_square(size, pad_ratio)
        img.save(out_path, "PNG", optimize=True)
        print(f"✓ {out_path.relative_to(ROOT)}  ({size}x{size})")


if __name__ == "__main__":
    main()
