package pdf

import "fmt"

// spell convierte un entero (positivo) a su representación en letras en
// español peruano. Cubre hasta 999.999.999, que excede cualquier monto
// realista de una factura MYPE.
func spell(n int64) string {
	if n < 0 {
		return "MENOS " + spell(-n)
	}
	if n == 0 {
		return "CERO"
	}
	if n >= 1_000_000 {
		mill := n / 1_000_000
		rem := n % 1_000_000
		var prefijo string
		if mill == 1 {
			prefijo = "UN MILLÓN"
		} else {
			prefijo = spell(mill) + " MILLONES"
		}
		if rem == 0 {
			return prefijo
		}
		return prefijo + " " + spell(rem)
	}
	if n >= 1000 {
		miles := n / 1000
		rem := n % 1000
		var prefijo string
		if miles == 1 {
			prefijo = "MIL"
		} else {
			prefijo = spell(miles) + " MIL"
		}
		if rem == 0 {
			return prefijo
		}
		return prefijo + " " + spell(rem)
	}
	if n >= 100 {
		c := n / 100
		rem := n % 100
		centenas := []string{"", "CIENTO", "DOSCIENTOS", "TRESCIENTOS", "CUATROCIENTOS",
			"QUINIENTOS", "SEISCIENTOS", "SETECIENTOS", "OCHOCIENTOS", "NOVECIENTOS"}
		if c == 1 && rem == 0 {
			return "CIEN"
		}
		if rem == 0 {
			return centenas[c]
		}
		return centenas[c] + " " + spell(rem)
	}
	if n >= 30 {
		d := n / 10
		u := n % 10
		decenas := []string{"", "", "VEINTE", "TREINTA", "CUARENTA", "CINCUENTA",
			"SESENTA", "SETENTA", "OCHENTA", "NOVENTA"}
		if u == 0 {
			return decenas[d]
		}
		return decenas[d] + " Y " + spell(u)
	}
	if n >= 20 {
		veintes := []string{"VEINTE", "VEINTIUNO", "VEINTIDÓS", "VEINTITRÉS", "VEINTICUATRO",
			"VEINTICINCO", "VEINTISÉIS", "VEINTISIETE", "VEINTIOCHO", "VEINTINUEVE"}
		return veintes[n-20]
	}
	if n >= 16 {
		dieces := []string{"DIECISÉIS", "DIECISIETE", "DIECIOCHO", "DIECINUEVE"}
		return dieces[n-16]
	}
	base := map[int64]string{
		0: "CERO", 1: "UNO", 2: "DOS", 3: "TRES", 4: "CUATRO", 5: "CINCO",
		6: "SEIS", 7: "SIETE", 8: "OCHO", 9: "NUEVE", 10: "DIEZ",
		11: "ONCE", 12: "DOCE", 13: "TRECE", 14: "CATORCE", 15: "QUINCE",
	}
	if v, ok := base[n]; ok {
		return v
	}
	return fmt.Sprintf("%d", n)
}
