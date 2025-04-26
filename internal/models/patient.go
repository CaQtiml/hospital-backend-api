package models

import "time"

type Patient struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	HospitalID   uint       `json:"hospital_id" gorm:"index"`
	PatientHN    string     `json:"patient_hn" gorm:"uniqueIndex:idx_hospital_hn"`
	FirstNameTH  string     `json:"first_name_th"`
	MiddleNameTH string     `json:"middle_name_th"`
	LastNameTH   string     `json:"last_name_th"`
	FirstNameEN  string     `json:"first_name_en"`
	MiddleNameEN string     `json:"middle_name_en"`
	LastNameEN   string     `json:"last_name_en"`
	DateOfBirth  *time.Time `json:"date_of_birth"` // Use pointer to handle potential nulls if needed
	NationalID   string     `json:"national_id" gorm:"index"`
	PassportID   string     `json:"passport_id" gorm:"index"`
	PhoneNumber  string     `json:"phone_number"`
	Email        string     `json:"email"`
	Gender       string     `json:"gender"` // "M", "F"
}

// PatientSearchQuery represents the query parameters for searching patients.
// Fields are pointers to distinguish between zero values (e.g., empty string) and fields not provided.
type PatientSearchQuery struct {
	NationalID   *string `form:"national_id"`
	PassportID   *string `form:"passport_id"`
	FirstNameTH  *string `form:"first_name_th"`
	FirstNameEN  *string `form:"first_name_en"`
	MiddleNameTH *string `form:"middle_name_th"`
	MiddleNameEN *string `form:"middle_name_en"`
	LastNameTH   *string `form:"last_name_th"`
	LastNameEN   *string `form:"last_name_en"`
	DateOfBirth  *string `form:"date_of_birth"` // Expecting YYYY-MM-DD format
	PhoneNumber  *string `form:"phone_number"`
	Email        *string `form:"email"`
}
