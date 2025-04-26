package database

import (
	"fmt"
	"hospital-middleware/internal/config"
	"hospital-middleware/internal/models"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database connection instance.
var DB *gorm.DB

// Connect initializes the database connection using GORM.
func Connect(cfg *config.Config) error {
	var err error
	log.Printf("Connecting to database %s on %s:%s...", cfg.DBName, cfg.DBHost, cfg.DBPort)
	log.Printf("DEBUG: Using configuration: %+v", cfg)
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Bangkok", // Adjust TimeZone if needed
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBPort,
		cfg.DBSSLMode,
	)

	// Configure GORM logger
	newLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level (Silent, Error, Warn, Info)
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Enable color
		},
	)

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connection successfully established")

	// Auto-migrate the schema
	// Create tables, columns, and indexes based on GORM models.
	log.Println("Running database migrations...")
	err = DB.AutoMigrate(&models.Staff{}, &models.Patient{})
	if err != nil {
		return fmt.Errorf("failed to auto-migrate database schema: %w", err)
	}
	log.Println("Database migrations completed.")

	return nil
}

// GetDB returns the initialized database connection instance.
func GetDB() *gorm.DB {
	return DB
}

// --- Staff Specific Functions ---

// CreateStaff inserts a new staff member into the database.
func CreateStaff(staff *models.Staff) error {
	result := DB.Create(staff)
	return result.Error
}

// FindStaffByUsername retrieves a staff member by their username.
func FindStaffByUsername(username string) (*models.Staff, error) {
	var staff models.Staff
	result := DB.Where("username = ?", username).First(&staff)
	if result.Error != nil {
		return nil, result.Error // Could be gorm.ErrRecordNotFound or other DB error
	}
	return &staff, nil
}

// --- Patient Specific Functions ---

func CreatePatient(patient *models.Patient) error {
	result := DB.Create(patient)
	return result.Error
}

// SearchPatients searches for patients based on criteria and hospital ID.
func SearchPatients(query *models.PatientSearchQuery, hospitalID uint) ([]models.Patient, error) {
	var patients []models.Patient
	dbQuery := DB.Model(&models.Patient{}).Where("hospital_id = ?", hospitalID)

	if query.NationalID != nil && *query.NationalID != "" {
		dbQuery = dbQuery.Where("national_id = ?", *query.NationalID)
	}
	if query.PassportID != nil && *query.PassportID != "" {
		dbQuery = dbQuery.Where("passport_id = ?", *query.PassportID)
	}

	// Handle First Name Search (Thai and English, Combined)
	if (query.FirstNameTH != nil && *query.FirstNameTH != "") && (query.FirstNameEN != nil && *query.FirstNameEN != "") {
		nameLikeTH := "%" + *query.FirstNameTH + "%"
		nameLikeEN := "%" + *query.FirstNameEN + "%"
		dbQuery = dbQuery.Where("first_name_th LIKE ? OR first_name_en LIKE ?", nameLikeTH, nameLikeEN)
	} else if query.FirstNameTH != nil && *query.FirstNameTH != "" {
		nameLike := "%" + *query.FirstNameTH + "%"
		dbQuery = dbQuery.Where("first_name_th LIKE ?", nameLike)
	} else if query.FirstNameEN != nil && *query.FirstNameEN != "" {
		nameLike := "%" + *query.FirstNameEN + "%"
		dbQuery = dbQuery.Where("first_name_en LIKE ?", nameLike)
	}

	// Handle Middle Name Search (Thai and English, Combined)
	if (query.MiddleNameTH != nil && *query.MiddleNameTH != "") && (query.MiddleNameEN != nil && *query.MiddleNameEN != "") {
		nameLikeTH := "%" + *query.MiddleNameTH + "%"
		nameLikeEN := "%" + *query.MiddleNameEN + "%"
		dbQuery = dbQuery.Where("middle_name_th LIKE ? OR middle_name_en LIKE ?", nameLikeTH, nameLikeEN)
	} else if query.MiddleNameTH != nil && *query.MiddleNameTH != "" {
		nameLike := "%" + *query.MiddleNameTH + "%"
		dbQuery = dbQuery.Where("middle_name_th LIKE ?", nameLike)
	} else if query.MiddleNameEN != nil && *query.MiddleNameEN != "" {
		nameLike := "%" + *query.MiddleNameEN + "%"
		dbQuery = dbQuery.Where("middle_name_en LIKE ?", nameLike)
	}

	// Handle Last Name Search (Thai and English, Combined)
	if (query.LastNameTH != nil && *query.LastNameTH != "") && (query.LastNameEN != nil && *query.LastNameEN != "") {
		nameLikeTH := "%" + *query.LastNameTH + "%"
		nameLikeEN := "%" + *query.LastNameEN + "%"
		dbQuery = dbQuery.Where("last_name_th LIKE ? OR last_name_en LIKE ?", nameLikeTH, nameLikeEN)
	} else if query.LastNameTH != nil && *query.LastNameTH != "" {
		nameLike := "%" + *query.LastNameTH + "%"
		dbQuery = dbQuery.Where("last_name_th LIKE ?", nameLike)
	} else if query.LastNameEN != nil && *query.LastNameEN != "" {
		nameLike := "%" + *query.LastNameEN + "%"
		dbQuery = dbQuery.Where("last_name_en LIKE ?", nameLike)
	}

	if query.DateOfBirth != nil && *query.DateOfBirth != "" {
		// Assuming YYYY-MM-DD format from query
		dob, err := time.Parse("2006-01-02", *query.DateOfBirth)
		if err == nil {
			dbQuery = dbQuery.Where("date_of_birth = ?", dob)
		} else {
			log.Printf("Warning: Invalid date format for date_of_birth: %s", *query.DateOfBirth)
		}
	}
	if query.PhoneNumber != nil && *query.PhoneNumber != "" {
		dbQuery = dbQuery.Where("phone_number = ?", *query.PhoneNumber)
	}
	if query.Email != nil && *query.Email != "" {
		dbQuery = dbQuery.Where("email = ?", *query.Email)
	}

	result := dbQuery.Find(&patients)
	if result.Error != nil {
		return nil, result.Error
	}

	return patients, nil
}

// --- Helper Function (Example for getting Hospital ID by Name) ---
func GetHospitalIDByName(hospitalName string) (uint, error) {
	// I know this is a stupid way to do it, but let it be for now. I want to finish the basic functionality first.
	if hospitalName == "Hospital A" { // Example
		return 1, nil
	}
	if hospitalName == "Hospital B" { // Example
		return 2, nil
	}
	// If no mapping found, return 0 or an error
	return 0, fmt.Errorf("hospital not found: %s", hospitalName)
}
