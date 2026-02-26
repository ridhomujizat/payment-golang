package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// JSONB is a custom type for GORM to handle JSONB columns
type JSONB json.RawMessage

func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = JSONB("null")
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*j = JSONB(v)
	case string:
		*j = JSONB(v)
	default:
		return errors.New("unsupported type for JSONB")
	}
	return nil
}

func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

func (j *JSONB) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("JSONB: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}

type Transaction struct {
	ID            string     `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID       string     `json:"order_id" gorm:"type:varchar(100);uniqueIndex;not null"`
	CustomerName  string     `json:"customer_name" gorm:"type:varchar(255)"`
	CustomerPhone string     `json:"customer_phone" gorm:"type:varchar(50)"`
	CustomerEmail string     `json:"customer_email" gorm:"type:varchar(255)"`
	GrossAmount   int64      `json:"gross_amount" gorm:"not null"`
	PaymentType   string     `json:"payment_type" gorm:"type:varchar(50)"`
	Items         JSONB      `json:"items" gorm:"type:jsonb;not null"`
	Metadata      JSONB      `json:"metadata" gorm:"type:jsonb"`
	SnapToken     string     `json:"snap_token" gorm:"type:varchar(255)"`
	SnapURL       string     `json:"snap_url" gorm:"type:text"`
	TransactionID string     `json:"transaction_id" gorm:"type:varchar(255)"`
	Status        string     `json:"status" gorm:"type:varchar(50);not null;default:'pending';index"`
	FraudStatus   string     `json:"fraud_status" gorm:"type:varchar(50)"`
	StatusCode    string     `json:"status_code" gorm:"type:varchar(10)"`
	SignatureKey  string     `json:"signature_key" gorm:"type:text"`
	CreatedAt     time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	PaidAt        *time.Time `json:"paid_at"`
}

func (Transaction) TableName() string {
	return "transactions"
}
