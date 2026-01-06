package organization

import "time"

type Organization struct {
	ID        int64     `json:"id"        gorm:"column:id;primaryKey"`
	Name      string    `json:"name"      gorm:"column:name"`
	Code      *string   `json:"code"      gorm:"column:code"`
	Active    bool      `json:"active"    gorm:"column:active"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
	DeletedAt time.Time `json:"deletedAt" gorm:"column:deleted_at"`
}

func (Organization) TableName() string {
	return "organizations"
}
