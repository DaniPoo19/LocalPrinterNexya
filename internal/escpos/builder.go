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

// EncodeCP850 convierte una cadena UTF-8 en bytes codificados en CodePage 850 (Latin-1 Multilingüe)
func EncodeCP850(s string) []byte {
	var buf bytes.Buffer
	for _, r := range s {
		switch r {
		case 'á':
			buf.WriteByte(0xA0)
		case 'é':
			buf.WriteByte(0x82)
		case 'í':
			buf.WriteByte(0xA1)
		case 'ó':
			buf.WriteByte(0xA2)
		case 'ú':
			buf.WriteByte(0xA3)
		case 'Á':
			buf.WriteByte(0xB5)
		case 'É':
			buf.WriteByte(0x90)
		case 'Í':
			buf.WriteByte(0xD6)
		case 'Ó':
			buf.WriteByte(0xE0)
		case 'Ú':
			buf.WriteByte(0xE9)
		case 'ñ':
			buf.WriteByte(0xA4)
		case 'Ñ':
			buf.WriteByte(0xA5)
		case 'ü':
			buf.WriteByte(0x81)
		case 'Ü':
			buf.WriteByte(0x9A)
		case '¿':
			buf.WriteByte(0xA8)
		case '¡':
			buf.WriteByte(0xAD)
		case '°':
			buf.WriteByte(0xF8)
		case '•', '·':
			buf.WriteByte(0xFA) // Punto medio en CP850
		default:
			if r <= 0x7F {
				buf.WriteByte(byte(r))
			} else {
				switch r {
				case '–', '—':
					buf.WriteByte('-')
				case '’', '‘':
					buf.WriteByte('\'')
				case '“', '”':
					buf.WriteByte('"')
				default:
					buf.WriteString(string(r))
				}
			}
		}
	}
	return buf.Bytes()
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
	b.buf.Write(CmdSelectCodePage850)
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

func (b *Builder) OpenDrawer() *Builder {
	b.buf.Write(CmdOpenDrawerPin2)
	b.buf.Write(CmdOpenDrawerPin5)
	return b
}

func (b *Builder) Beep() *Builder {
	b.buf.Write(CmdBeep)
	return b
}

func (b *Builder) Cut(partial bool) *Builder {
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
