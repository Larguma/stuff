package models

import "time"

type User struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"uniqueIndex"`
	PasswordHash string
	CreatedAt    time.Time
}

type House struct {
	ID        uint `gorm:"primaryKey"`
	Name      string
	CreatedAt time.Time
}

type HouseMember struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"index:idx_house_member,unique"`
	HouseID   uint `gorm:"index:idx_house_member,unique"`
	CreatedAt time.Time
}

type Invite struct {
	ID              uint   `gorm:"primaryKey"`
	HouseID         uint   `gorm:"index"`
	Code            string `gorm:"uniqueIndex"`
	CreatedByUserID uint
	UsedByUserID    *uint
	UsedAt          *time.Time
	CreatedAt       time.Time
}

type Location struct {
	ID             uint `gorm:"primaryKey"`
	HouseID        uint `gorm:"index:idx_location_house_name,unique"`
	Name           string
	NameNormalized string `gorm:"index:idx_location_house_name,unique"`
	CreatedAt      time.Time
}

type Tag struct {
	ID             uint `gorm:"primaryKey"`
	HouseID        uint `gorm:"index:idx_tag_house_name,unique"`
	Name           string
	NameNormalized string `gorm:"index:idx_tag_house_name,unique"`
	CreatedAt      time.Time
}

type Item struct {
	ID              uint `gorm:"primaryKey"`
	HouseID         uint `gorm:"index"`
	Name            string
	Notes           string `gorm:"type:text"`
	Quantity        int
	LocationID      *uint `gorm:"index"`
	Link            string
	CreatedByUserID uint
	CreatedAt       time.Time
	UpdatedAt       time.Time

	Location Location
	Tags     []Tag `gorm:"many2many:item_tags;"`
	Images   []ItemImage
}

type ItemImage struct {
	ID           uint `gorm:"primaryKey"`
	ItemID       uint `gorm:"index"`
	Path         string
	OriginalName string
	CreatedAt    time.Time
}
