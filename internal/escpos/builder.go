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

func cleanAscii(s string) string {
	replacer := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U",
		"ñ", "n", "Ñ", "N",
		"ü", "u", "Ü", "U",
		"¡", "", "¿", "",
		"–", "-", "—", "-", "’", "'", "”", "\"", "“", "\"",
	)
	return replacer.Replace(s)
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
	cleaned := cleanAscii(text)
	b.buf.WriteString(cleaned)
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
	b.buf.WriteString(repeated)
	b.buf.Write(CmdLineFeed)
	return b
}

// PrintRow2Cols imprime 2 columnas justificadas a los extremos (ej: "Total" a la izquierda, "$ 25.000" a la derecha)
func (b *Builder) PrintRow2Cols(left, right string) *Builder {
	leftClean := cleanAscii(left)
	rightClean := cleanAscii(right)

	leftLen := utf8.RuneCountInString(leftClean)
	rightLen := utf8.RuneCountInString(rightClean)
	totalWidth := b.lineWidth

	spaceCount := totalWidth - leftLen - rightLen
	if spaceCount < 1 {
		maxLeft := totalWidth - rightLen - 1
		if maxLeft > 3 {
			runes := []rune(leftClean)
			if len(runes) > maxLeft {
				leftClean = string(runes[:maxLeft])
				leftLen = maxLeft
			}
		}
		spaceCount = totalWidth - leftLen - rightLen
		if spaceCount < 1 {
			spaceCount = 1
		}
	}

	line := leftClean + strings.Repeat(" ", spaceCount) + rightClean
	b.buf.WriteString(line)
	b.buf.Write(CmdLineFeed)
	return b
}

// PrintItemRow imprime producto con cantidad, nombre y precio ajustado al ancho de papel
func (b *Builder) PrintItemRow(qty int, name string, priceStr string) *Builder {
	qtyStr := fmt.Sprintf("%dx ", qty)
	nameClean := cleanAscii(name)
	priceClean := cleanAscii(priceStr)

	qtyLen := utf8.RuneCountInString(qtyStr)
	priceLen := utf8.RuneCountInString(priceClean)

	maxNameLen := b.lineWidth - qtyLen - priceLen - 1
	if maxNameLen < 4 {
		maxNameLen = 4
	}

	nameRunes := []rune(nameClean)
	if len(nameRunes) > maxNameLen {
		nameClean = string(nameRunes[:maxNameLen])
	}

	left := qtyStr + nameClean
	b.PrintRow2Cols(left, priceClean)
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
