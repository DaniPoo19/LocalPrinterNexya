package escpos

import (
	"bytes"
	"testing"
)

func TestEncodeCP850SpanishCharacters(t *testing.T) {
	input := "¡Heladería con Limón, Café y Piña! 100° – 'Especial'"
	encoded := EncodeCP850(input)

	// Validate non-empty
	if len(encoded) == 0 {
		t.Fatal("Encoded bytes is empty")
	}

	// Validate ¡ is 0xAD
	if !bytes.Contains(encoded, []byte{0xAD}) {
		t.Errorf("Expected encoded output to contain 0xAD for '¡'")
	}

	// Validate í is 0xA1
	if !bytes.Contains(encoded, []byte{0xA1}) {
		t.Errorf("Expected encoded output to contain 0xA1 for 'í'")
	}

	// Validate é is 0x82
	if !bytes.Contains(encoded, []byte{0x82}) {
		t.Errorf("Expected encoded output to contain 0x82 for 'é'")
	}

	// Validate ñ is 0xA4
	if !bytes.Contains(encoded, []byte{0xA4}) {
		t.Errorf("Expected encoded output to contain 0xA4 for 'ñ'")
	}

	// Validate ° is 0xF8
	if !bytes.Contains(encoded, []byte{0xF8}) {
		t.Errorf("Expected encoded output to contain 0xF8 for '°'")
	}
}
