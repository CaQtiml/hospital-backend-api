package models

import "time"

// Staff represents the hospital staff data model.
type Staff struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"uniqueIndex;not null"` // Unique username for login
	PasswordHash string    `json:"-" gorm:"not null"`                    // "-" prevents it from being marshalled into JSON
	HospitalID   uint      `json:"hospital_id" gorm:"index;not null"`    // ID of the hospital the staff belongs to
	HospitalName string    `json:"hospital_name" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"not null"`
	UpdatedAt    time.Time `json:"updated_at " gorm:"not null"`
}

// StaffCreateRequest represents the input for creating a new staff member.
type StaffCreateRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Hospital string `json:"hospital" binding:"required"` // Hospital Name or ID
}

// StaffLoginRequest represents the input for staff login.
type StaffLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Hospital string `json:"hospital" binding:"required"` // Hospital Name or ID
}

// StaffLoginResponse represents the output after successful login.
type StaffLoginResponse struct {
	Token string `json:"token"`
	Staff Staff  `json:"staff"` // Return basic staff info (excluding password)
}
