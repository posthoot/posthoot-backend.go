package models

import (
	"time"
)

type User struct {
	Base
	Email       string           `gorm:"uniqueIndex;not null" json:"email"`
	Password    string           `gorm:"not null" json:"-"`
	FirstName   string           `json:"firstName"`
	LastName    string           `json:"lastName"`
	Role        UserRole         `gorm:"not null;default:'member'" json:"role"`
	TeamID      string           `gorm:"type:uuid;not null" json:"teamId"`
	Team        *Team            `json:"team,omitempty"`
	Permissions []UserPermission `gorm:"foreignKey:UserID" json:"permissions,omitempty"`
	Invites     []TeamInvite     `gorm:"foreignKey:InviterID" json:"invites,omitempty"`
	Files       []File           `gorm:"foreignKey:UserID" json:"files,omitempty"`
}

type PasswordReset struct {
	Base
	User      *User     `json:"user,omitempty"`
	UserID    string    `gorm:"type:uuid;not null" json:"userId"`
	Code      string    `gorm:"not null" json:"code"`
	Used      bool      `gorm:"default:false" json:"used"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type AuthTransaction struct {
	Base
	UserID    string    `gorm:"type:uuid;not null" json:"userId"`
	User      *User     `json:"user,omitempty"`
	TeamID    string    `gorm:"type:uuid;not null" json:"teamId"`
	Team      *Team     `json:"team,omitempty"`
	Token     string    `gorm:"not null" json:"token"`
	Refresh   string    `gorm:"not null" json:"refresh"`
	IPAddress string    `json:"ipAddress"`
	UserAgent string    `json:"userAgent"`
	ExpiresAt time.Time `json:"expiresAt"`
}
