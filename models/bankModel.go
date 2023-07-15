package models

type Bank struct {
	Id     uint   `gorm:"primaryKey:autoIncrement"`
	UserId int64  `json:"userId"`
	User   User   `gorm:"foreignKey:UserId;references:Id"`
	Name   string `json:"name"`
	Icon   string `json:"icon"`
}

func (bank *Bank) TableName() string {
	return "bank"
}
