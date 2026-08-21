package escpos

var (
	// Control
	CmdInit           = []byte{0x1B, 0x40}                   // ESC @ (Reset/Init)
	CmdLineFeed       = []byte{0x0A}                         // LF (Line feed)
	
	// Alignment
	CmdAlignLeft      = []byte{0x1B, 0x61, 0x00}             // ESC a 0
	CmdAlignCenter    = []byte{0x1B, 0x61, 0x01}             // ESC a 1
	CmdAlignRight     = []byte{0x1B, 0x61, 0x02}             // ESC a 2

	// Font styles
	CmdBoldOn         = []byte{0x1B, 0x45, 0x01}             // ESC E 1 (Bold ON)
	CmdBoldOff        = []byte{0x1B, 0x45, 0x00}             // ESC E 0 (Bold OFF)
	CmdUnderlineOn    = []byte{0x1B, 0x2D, 0x01}             // ESC - 1
	CmdUnderlineOff   = []byte{0x1B, 0x2D, 0x00}             // ESC - 0
	CmdInvertOn       = []byte{0x1D, 0x42, 0x01}             // GS B 1 (White on Black)
	CmdInvertOff      = []byte{0x1D, 0x42, 0x00}             // GS B 0

	// Text sizes
	CmdSizeNormal     = []byte{0x1D, 0x21, 0x00}             // GS ! 0 (Normal 1x1)
	CmdSizeDoubleW    = []byte{0x1D, 0x21, 0x20}             // GS ! 32 (Double width)
	CmdSizeDoubleH    = []byte{0x1D, 0x21, 0x01}             // GS ! 1 (Double height)
	CmdSizeDoubleAll  = []byte{0x1D, 0x21, 0x11}             // GS ! 17 (Double width & height)
	CmdFontB          = []byte{0x1B, 0x4D, 0x01}             // ESC M 1 (Small font B)
	CmdFontA          = []byte{0x1B, 0x4D, 0x00}             // ESC M 0 (Standard font A)

	// Paper Cut
	CmdCutFull        = []byte{0x1D, 0x56, 0x00}             // GS V 0 (Full Cut)
	CmdCutPartial     = []byte{0x1D, 0x56, 0x01}             // GS V 1 (Partial Cut)
	CmdFeedAndCut     = []byte{0x1D, 0x56, 0x41, 0x03}       // GS V 65 3 (Feed 3 lines + Partial Cut)

	// Hardware Actions (Cash Drawer / Gaveta Monedero)
	// ESC p estándar internacional (Pin 2 y Pin 5 con pulso 50ms)
	CmdOpenDrawerPin2 = []byte{0x1B, 0x70, 0x00, 0x19, 0xFA} // ESC p 0 25 250 (Pin 2 RJ12)
	CmdOpenDrawerPin5 = []byte{0x1B, 0x70, 0x01, 0x19, 0xFA} // ESC p 1 25 250 (Pin 5 RJ12)

	// Secuencia universal limpia y segura (Pin 2 y Pin 5 estándar)
	CmdOpenDrawerUniversal = []byte{
		0x1B, 0x70, 0x00, 0x19, 0xFA, // Pin 2
		0x1B, 0x70, 0x01, 0x19, 0xFA, // Pin 5
	}

	CmdBeep = []byte{0x1B, 0x42, 0x02, 0x02} // ESC B 2 2 (Beep buzzer 2 times)

	// CodePage Selection
	CmdSelectCodePage437  = []byte{0x1B, 0x74, 0x00} // ESC t 0 (CodePage 437 Standard USA / Latin)
	CmdSelectCodePage850  = []byte{0x1B, 0x74, 0x02} // ESC t 2 (CodePage 850 Latin-1)
	CmdSelectCodePage1252 = []byte{0x1B, 0x74, 0x10} // ESC t 16 (CodePage 1252 Windows Latin-1)

	// Line Spacing
	CmdLineSpacing36 = []byte{0x1B, 0x33, 0x24} // ESC 3 36 (36 dots line spacing = 1.2 line height)
	CmdLineSpacing24 = []byte{0x1B, 0x33, 0x18} // ESC 3 24 (Default 24 dots line spacing)
)
