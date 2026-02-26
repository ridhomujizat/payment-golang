package payment

// WAFlowRequest represents the incoming request from WhatsApp Flow
type WAFlowRequest struct {
	Action string          `json:"action"`
	Data   WAFlowOrderData `json:"data"`
}

// WAFlowResponse represents the response sent back to WhatsApp Flow
type WAFlowResponse struct {
	Screen string          `json:"screen"`
	Data   WAFlowOrderData `json:"data"`
}

// WAFlowOrderData contains order form fields exchanged with WhatsApp Flow
type WAFlowOrderData struct {
	NamaPenerima    string `json:"nama_penerima"`
	NomorHandphone  string `json:"nomor_handphone"`
	AlamatLengkap   string `json:"alamat_lengkap"`
	Provinsi        string `json:"provinsi"`
	KotaKecamatan   string `json:"kota_kecamatan"`
	KodePos         string `json:"kode_pos"`
	ItemsText       string `json:"items_text"`
	TotalBarang     string `json:"total_barang"`
	TotalPengiriman string `json:"total_pengiriman"`
	TotalBiaya      string `json:"total_biaya"`
}
