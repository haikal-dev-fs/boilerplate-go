package device

// ========= DTO REQUEST & RESPONSE =========

type CreateDataSourceRequest struct {
	Name string `json:"name"`
	Code string `json:"code"`
	Type string `json:"type"` // contoh: "TELTONIKA", "CARTRACK_API", dll
}

type CreateDeviceRequest struct {
	DataSourceID int64                  `json:"dataSourceId"`
	ExternalID   string                 `json:"externalId"` // biasanya IMEI / device ID dari vendor
	SimNumber    string                 `json:"simNumber"`
	Model        string                 `json:"model"`
	Protocol     string                 `json:"protocol"`
	Metadata     map[string]interface{} `json:"metadata"` // optional, bisa kosong
}

type UpdateDataSourceRequest struct {
	Name *string `json:"name,omitempty"`
	Code *string `json:"code,omitempty"`
	Type *string `json:"type,omitempty"`
}

type UpdateDeviceRequest struct {
	DataSourceID *int64                 `json:"dataSourceId,omitempty"`
	SimNumber    *string                `json:"simNumber,omitempty"`
	Model        *string                `json:"model,omitempty"`
	Protocol     *string                `json:"protocol,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Active       *bool                  `json:"active,omitempty"`
}
