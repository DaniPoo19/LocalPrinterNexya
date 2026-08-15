package escpos

import (
	"fmt"
	"strings"
	"time"
)

type TicketProduct struct {
	Name           string   `json:"name"`
	Quantity       int      `json:"quantity"`
	Price          float64  `json:"price"`
	Observation    string   `json:"observation,omitempty"`
	Customizations []string `json:"customizations,omitempty"`
}

type PrintOrderRequest struct {
	PrinterName    string          `json:"printer_name,omitempty"`
	PaperWidth     string          `json:"paper_width,omitempty"` // "58mm" or "80mm"
	BusinessName   string          `json:"business_name,omitempty"`
	NIT            string          `json:"nit,omitempty"`
	Address        string          `json:"address,omitempty"`
	Phone          string          `json:"phone,omitempty"`
	OrderCode      string          `json:"order_code"`
	DailyCode      string          `json:"daily_code,omitempty"`
	CreatedAt      string          `json:"created_at,omitempty"`
	SaleType       string          `json:"sale_type,omitempty"` // "ON_SITE", "DELIVERY", "PICKUP"
	WaiterName     string          `json:"waiter_name,omitempty"`
	CustomerName   string          `json:"customer_name,omitempty"`
	CustomerPhone  string          `json:"customer_phone,omitempty"`
	DeliveryAddress string         `json:"delivery_address,omitempty"`
	Products       []TicketProduct `json:"products"`
	Subtotal       float64         `json:"subtotal"`
	DeliveryCost   float64         `json:"delivery_cost,omitempty"`
	Discount       float64         `json:"discount,omitempty"`
	Total          float64         `json:"total"`
	PaymentType    string          `json:"payment_type,omitempty"`
	CashAmount     float64         `json:"cash_amount,omitempty"`
	TransferAmount float64         `json:"transfer_amount,omitempty"`
	ChangeAmount   float64         `json:"change_amount,omitempty"`
	FooterMessage  string          `json:"footer_message,omitempty"`
	OpenDrawer     bool            `json:"open_drawer"`
	CutPaper       bool            `json:"cut_paper"`
	Beep           bool            `json:"beep"`
	Copies         int             `json:"copies,omitempty"`
}

func FormatOrderTicket(req *PrintOrderRequest) []byte {
	paperWidth := req.PaperWidth
	if paperWidth == "" {
		paperWidth = "80mm"
	}

	b := NewBuilder(paperWidth)

	if req.Beep {
		b.Beep()
	}

	if req.OpenDrawer {
		b.OpenDrawer()
	}

	// 1. ENCABEZADO EMPRESA
	b.SetAlign("center")
	if req.BusinessName != "" {
		b.SetBold(true).SetDoubleHeight(true)
		b.PrintLine(req.BusinessName)
		b.SetBold(false).SetDoubleHeight(false)
	}

	if req.NIT != "" {
		b.PrintLine(fmt.Sprintf("NIT: %s", req.NIT))
	}
	if req.Address != "" {
		b.PrintLine(req.Address)
	}
	if req.Phone != "" {
		b.PrintLine(fmt.Sprintf("Tel: %s", req.Phone))
	}

	b.PrintDivider("=")

	// 2. DETALLES DEL PEDIDO
	b.SetAlign("center")
	b.SetBold(true).SetDoubleSize(true)
	displayCode := req.OrderCode
	if req.DailyCode != "" {
		displayCode = req.DailyCode
	}
	b.PrintLine(fmt.Sprintf("PEDIDO #%s", displayCode))
	b.SetBold(false).SetDoubleSize(false)

	b.SetAlign("left")
	dateStr := req.CreatedAt
	if dateStr == "" {
		dateStr = time.Now().Format("02/01/2006 15:04")
	}
	b.PrintRow2Cols("Fecha:", dateStr)

	saleTypeLabel := "En Local"
	if strings.EqualFold(req.SaleType, "DELIVERY") {
		saleTypeLabel = "DOMICILIO"
	} else if strings.EqualFold(req.SaleType, "PICKUP") {
		saleTypeLabel = "PARA RECOGER"
	}
	b.PrintRow2Cols("Tipo:", saleTypeLabel)

	if req.WaiterName != "" {
		b.PrintRow2Cols("Atendido por:", req.WaiterName)
	}

	// Datos del cliente
	if req.CustomerName != "" {
		b.PrintDivider("-")
		b.SetBold(true).PrintLine("CLIENTE:")
		b.SetBold(false)
		b.PrintLine(fmt.Sprintf(" Nombre: %s", req.CustomerName))
		if req.CustomerPhone != "" {
			b.PrintLine(fmt.Sprintf(" Tel:    %s", req.CustomerPhone))
		}
		if req.DeliveryAddress != "" {
			b.PrintLine(fmt.Sprintf(" Dir:    %s", req.DeliveryAddress))
		}
	}

	b.PrintDivider("=")

	// 3. PRODUCTOS
	b.SetBold(true)
	b.PrintRow2Cols("CANT. PRODUCTO", "VALOR")
	b.SetBold(false)
	b.PrintDivider("-")

	for _, p := range req.Products {
		priceStr := formatCurrency(p.Price * float64(p.Quantity))
		b.PrintItemRow(p.Quantity, p.Name, priceStr)

		// Personalizaciones / Sabores
		for _, cust := range p.Customizations {
			b.SetSmallFont(true)
			b.PrintLine(fmt.Sprintf("  + %s", cust))
			b.SetSmallFont(false)
		}

		// Observaciones
		if p.Observation != "" {
			b.SetSmallFont(true)
			b.PrintLine(fmt.Sprintf("  * NOTA: %s", p.Observation))
			b.SetSmallFont(false)
		}
	}

	b.PrintDivider("-")

	// 4. TOTALES
	if req.Subtotal > 0 && (req.DeliveryCost > 0 || req.Discount > 0) {
		b.PrintRow2Cols("Subtotal:", formatCurrency(req.Subtotal))
	}
	if req.DeliveryCost > 0 {
		b.PrintRow2Cols("Domicilio:", formatCurrency(req.DeliveryCost))
	}
	if req.Discount > 0 {
		b.PrintRow2Cols("Descuento:", fmt.Sprintf("-%s", formatCurrency(req.Discount)))
	}

	b.SetBold(true).SetDoubleHeight(true)
	b.PrintRow2Cols("TOTAL:", formatCurrency(req.Total))
	b.SetBold(false).SetDoubleHeight(false)

	// 5. DETALLES DE PAGO
	if req.PaymentType != "" {
		b.PrintDivider("-")
		b.PrintRow2Cols("Metodo de Pago:", req.PaymentType)
		if req.CashAmount > 0 {
			b.PrintRow2Cols("Efectivo:", formatCurrency(req.CashAmount))
		}
		if req.TransferAmount > 0 {
			b.PrintRow2Cols("Transferencia:", formatCurrency(req.TransferAmount))
		}
		if req.ChangeAmount > 0 {
			b.PrintRow2Cols("Cambio:", formatCurrency(req.ChangeAmount))
		}
	}

	// 6. PIE DE PÁGINA
	b.PrintDivider("=")
	b.SetAlign("center")
	footer := req.FooterMessage
	if footer == "" {
		footer = "¡Gracias por su compra!\nVuelva pronto"
	}
	for _, line := range strings.Split(footer, "\n") {
		if strings.TrimSpace(line) != "" {
			b.PrintLine(line)
		}
	}

	// 7. CORTE
	if req.CutPaper {
		b.Cut(true)
	} else {
		b.FeedLines(4)
	}

	return b.Bytes()
}

func FormatTestTicket(paperWidth string) []byte {
	if paperWidth == "" {
		paperWidth = "80mm"
	}
	b := NewBuilder(paperWidth)
	b.SetAlign("center")
	b.SetBold(true).SetDoubleSize(true)
	b.PrintLine("NEXYA PRINTER")
	b.SetBold(false).SetDoubleSize(false)
	b.PrintLine("Impresion de Diagnostico")
	b.PrintDivider("=")
	b.SetAlign("left")
	b.PrintRow2Cols("Estado:", "OK - Conectado")
	b.PrintRow2Cols("Ancho:", paperWidth)
	b.PrintRow2Cols("Fecha:", time.Now().Format("02/01/2006 15:04:05"))
	b.PrintDivider("-")
	b.SetAlign("center")
	b.PrintLine("Hardware Spooler: Operativo")
	b.PrintLine("Corte de papel: Activo")
	b.PrintDivider("=")
	b.PrintLine("Sistema Listo para Operar")
	b.Cut(true)
	return b.Bytes()
}

func formatCurrency(val float64) string {
	// Formatear como moneda colombiana (ej: $ 25.000)
	intVal := int64(val)
	str := fmt.Sprintf("%d", intVal)
	var parts []string
	for len(str) > 3 {
		parts = append([]string{str[len(str)-3:]}, parts...)
		str = str[:len(str)-3]
	}
	if len(str) > 0 {
		parts = append([]string{str}, parts...)
	}
	return fmt.Sprintf("$ %s", strings.Join(parts, "."))
}
