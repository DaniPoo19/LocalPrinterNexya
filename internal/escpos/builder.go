package escpos

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"
)

type Builder struct {
	buf       bytes.Buffer
	lineWidth int // 32 for 58mm, 48 for 80mm
}

// EncodeSafeText convierte una cadena UTF-8 en bytes estándar ESC/POS (CP437/ASCII seguro)
// Sanitiza caracteres especiales y símbolos para garantizar que NINGÚN carácter corrupto o japonés
// pueda imprimirse jamás en impresoras genéricas POS-58 o POS-80.
func EncodeSafeText(s string) []byte {
	var buf bytes.Buffer
	for _, r := range s {
		switch r {
		case 'á', 'à', 'ä', 'â', 'ã':
			buf.WriteByte(0xA0) // á en CP437
		case 'é', 'è', 'ë', 'ê':
			buf.WriteByte(0x82) // é en CP437
		case 'í', 'ì', 'ï', 'î':
			buf.WriteByte(0xA1) // í en CP437
		case 'ó', 'ò', 'ö', 'ô', 'õ':
			buf.WriteByte(0xA2) // ó en CP437
		case 'ú', 'ù', 'û':
			buf.WriteByte(0xA3) // ú en CP437
		case 'Á', 'À', 'Ä', 'Â', 'Ã':
			buf.WriteByte('A') // Á seguro
		case 'É', 'È', 'Ë', 'Ê':
			buf.WriteByte(0x90) // É en CP437
		case 'Í', 'Ì', 'Ï', 'Î':
			buf.WriteByte('I') // Í seguro
		case 'Ó', 'Ò', 'Ö', 'Ô', 'Õ':
			buf.WriteByte('O') // Ó seguro
		case 'Ú', 'Ù', 'Û':
			buf.WriteByte('U') // Ú seguro
		case 'ñ':
			buf.WriteByte(0xA4) // ñ en CP437
		case 'Ñ':
			buf.WriteByte(0xA5) // Ñ en CP437
		case 'ü':
			buf.WriteByte(0x81) // ü en CP437
		case 'Ü':
			buf.WriteByte(0x9A) // Ü en CP437
		case '¿':
			buf.WriteByte(0xA8) // ¿ en CP437
		case '¡':
			buf.WriteByte(0xAD) // ¡ en CP437
		case '°', 'º', 'ª':
			buf.WriteByte(0xF8) // ° en CP437
		case '•', '·', '●', '▪', '◆', '▸', '►':
			buf.WriteByte('*') // Viñeta segura como asterisco
		case '×':
			buf.WriteByte('x')
		case '–', '—', '−':
			buf.WriteByte('-')
		case '’', '‘', '`', '´':
			buf.WriteByte('\'')
		case '“', '”', '«', '»':
			buf.WriteByte('"')
		case '…':
			buf.WriteString("...")
		default:
			if r >= 32 && r <= 126 {
				buf.WriteByte(byte(r))
			} else if r == '\n' || r == '\r' || r == '\t' {
				buf.WriteByte(byte(r))
			} else {
				// Carácter no soportado: reemplazar con espacio
				buf.WriteByte(' ')
			}
		}
	}
	return buf.Bytes()
}

// EncodeCP850 alias para compatibilidad interna
func EncodeCP850(s string) []byte {
	return EncodeSafeText(s)
}

func NewBuilder(paperWidth string) *Builder {
	width := 48
	if strings.EqualFold(strings.TrimSpace(paperWidth), "58mm") {
		width = 32
	}
	b := &Builder{lineWidth: width}
	b.Init()
	return b
}

func (b *Builder) Init() *Builder {
	b.buf.Write(CmdInit)
	b.buf.Write(CmdSelectCodePage437)
	b.buf.Write(CmdLineSpacing36)
	return b
}

func (b *Builder) SetLineSpacing(dots int) *Builder {
	if dots <= 0 {
		b.buf.Write(CmdLineSpacing24)
	} else {
		b.buf.Write([]byte{0x1B, 0x33, byte(dots)})
	}
	return b
}

func (b *Builder) SetAlign(align string) *Builder {
	switch strings.ToLower(align) {
	case "center":
		b.buf.Write(CmdAlignCenter)
	case "right":
		b.buf.Write(CmdAlignRight)
	default:
		b.buf.Write(CmdAlignLeft)
	}
	return b
}

func (b *Builder) SetBold(enable bool) *Builder {
	if enable {
		b.buf.Write(CmdBoldOn)
	} else {
		b.buf.Write(CmdBoldOff)
	}
	return b
}

func (b *Builder) SetDoubleSize(enable bool) *Builder {
	if enable {
		b.buf.Write(CmdSizeDoubleAll)
	} else {
		b.buf.Write(CmdSizeNormal)
	}
	return b
}

func (b *Builder) SetDoubleHeight(enable bool) *Builder {
	if enable {
		b.buf.Write(CmdSizeDoubleH)
	} else {
		b.buf.Write(CmdSizeNormal)
	}
	return b
}

func (b *Builder) SetSmallFont(enable bool) *Builder {
	if enable {
		b.buf.Write(CmdFontB)
	} else {
		b.buf.Write(CmdFontA)
	}
	return b
}

func (b *Builder) PrintLine(text string) *Builder {
	b.buf.Write(EncodeCP850(text))
	b.buf.Write(CmdLineFeed)
	return b
}

func (b *Builder) FeedLines(n int) *Builder {
	for i := 0; i < n; i++ {
		b.buf.Write(CmdLineFeed)
	}
	return b
}

func (b *Builder) PrintDivider(char string) *Builder {
	if char == "" {
		char = "-"
	}
	repeated := strings.Repeat(char, b.lineWidth)
	b.buf.Write(EncodeCP850(repeated))
	b.buf.Write(CmdLineFeed)
	return b
}

// PrintDoubleSizeRow2Cols imprime 2 columnas en Doble Tamaño Completo (letras grandes de 5mm, 24 caracteres por línea)
func (b *Builder) PrintDoubleSizeRow2Cols(left, right string) *Builder {
	leftLen := utf8.RuneCountInString(left)
	rightLen := utf8.RuneCountInString(right)
	totalWidth := b.lineWidth / 2 // 24 caracteres para 80mm

	spaceCount := totalWidth - leftLen - rightLen
	if spaceCount < 1 {
		maxLeft := totalWidth - rightLen - 1
		if maxLeft > 3 {
			runes := []rune(left)
			if len(runes) > maxLeft {
				left = string(runes[:maxLeft])
				leftLen = maxLeft
			}
		}
		spaceCount = totalWidth - leftLen - rightLen
		if spaceCount < 1 {
			spaceCount = 1
		}
	}

	line := left + strings.Repeat(" ", spaceCount) + right
	b.SetBold(true).SetDoubleSize(true)
	b.buf.Write(EncodeCP850(line))
	b.buf.Write(CmdLineFeed)
	b.SetBold(false).SetDoubleSize(false)
	return b
}

// PrintItemRowDoubleHeight imprime el producto y precio en Doble Altura Negrita (3.5mm de alto, igual a Chrome)
func (b *Builder) PrintItemRowDoubleHeight(qty int, name string, priceStr string) *Builder {
	b.SetBold(true).SetDoubleHeight(true)
	b.PrintItemRow(qty, name, priceStr)
	b.SetBold(false).SetDoubleHeight(false)
	return b
}

// PrintRow2Cols imprime 2 columnas justificadas a los extremos (ej: "Total" a la izquierda, "$ 25.000" a la derecha)
func (b *Builder) PrintRow2Cols(left, right string) *Builder {
	leftLen := utf8.RuneCountInString(left)
	rightLen := utf8.RuneCountInString(right)
	totalWidth := b.lineWidth

	spaceCount := totalWidth - leftLen - rightLen
	if spaceCount < 1 {
		maxLeft := totalWidth - rightLen - 1
		if maxLeft > 3 {
			runes := []rune(left)
			if len(runes) > maxLeft {
				left = string(runes[:maxLeft])
				leftLen = maxLeft
			}
		}
		spaceCount = totalWidth - leftLen - rightLen
		if spaceCount < 1 {
			spaceCount = 1
		}
	}

	line := left + strings.Repeat(" ", spaceCount) + right
	b.buf.Write(EncodeCP850(line))
	b.buf.Write(CmdLineFeed)
	return b
}

// PrintItemRow imprime producto con cantidad, nombre y precio ajustado al ancho de papel
func (b *Builder) PrintItemRow(qty int, name string, priceStr string) *Builder {
	qtyStr := fmt.Sprintf("%dx ", qty)
	qtyLen := utf8.RuneCountInString(qtyStr)
	priceLen := utf8.RuneCountInString(priceStr)

	maxNameLen := b.lineWidth - qtyLen - priceLen - 1
	if maxNameLen < 4 {
		maxNameLen = 4
	}

	nameRunes := []rune(name)
	if len(nameRunes) > maxNameLen {
		name = string(nameRunes[:maxNameLen])
	}

	left := qtyStr + name
	b.PrintRow2Cols(left, priceStr)
	return b
}

// BuildDrawerKickBytes retorna la secuencia pura de apertura de cajón monedero
// SIN comandos de reinicio (ESC @) ni cambios de página que puedan interferir en el controlador RJ11
func BuildDrawerKickBytes() []byte {
	return CmdOpenDrawerUniversal
}

func (b *Builder) OpenDrawer() *Builder {
	b.buf.Write(CmdOpenDrawerUniversal)
	return b
}

func (b *Builder) Beep() *Builder {
	b.buf.Write(CmdBeep)
	return b
}

func (b *Builder) Cut(partial bool) *Builder {
	if b.lineWidth <= 32 {
		// En 58mm las impresoras son de rasgado manual; avanzar líneas limpias para corte manual
		b.FeedLines(5)
		return b
	}

	b.FeedLines(3)
	if partial {
		b.buf.Write(CmdCutPartial)
	} else {
		b.buf.Write(CmdCutFull)
	}
	return b
}

func (b *Builder) PrintRasterImage(data []byte) *Builder {
	if len(data) > 0 {
		b.buf.Write(data)
	}
	return b
}

func (b *Builder) PrintRawBytes(data []byte) *Builder {
	if len(data) > 0 {
		b.buf.Write(data)
	}
	return b
}

func (b *Builder) Bytes() []byte {
	return b.buf.Bytes()
}
