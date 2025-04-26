package services

import (
	"errors"
	"fmt"
	"hospital-middleware/internal/config"
	"hospital-middleware/internal/database"
	"hospital-middleware/internal/models"
	"hospital-middleware/pkg/utils"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// Claims defines the structure of the JWT claims.
type Claims struct {
	UserID     uint   `json:"user_id"`
	Username   string `json:"username"`
	HospitalID uint   `json:"hospital_id"`
	jwt.RegisteredClaims
}

// Package-level variables to store config loaded during initialization
var (
	jwtKey    []byte
	jwtExpiry time.Duration
)

// InitializeAuthService sets up the JWT secret key and expiry duration.
func InitializeAuthService(cfg *config.Config) {
	jwtKey = []byte(cfg.JWTSecret)
	jwtExpiry = cfg.JWTExpiry // Store the expiry duration
	log.Printf("Auth service initialized with JWT expiry: %v", jwtExpiry)
}

// AuthenticateStaff checks staff credentials and generates a JWT token upon success.
func AuthenticateStaff(loginReq models.StaffLoginRequest) (string, *models.Staff, error) {
	// 1. Find the staff member by username
	staff, err := database.FindStaffByUsername(loginReq.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Authentication failed: User not found - %s", loginReq.Username)
			return "", nil, errors.New("invalid username or password")
		}
		log.Printf("Database error during login for user %s: %v", loginReq.Username, err)
		return "", nil, fmt.Errorf("database error during login: %w", err)
	}

	// 2. Check if the provided hospital matches the staff's hospital
	inputHospitalID, err := database.GetHospitalIDByName(loginReq.Hospital)
	if err != nil {
		log.Printf("Authentication failed: Hospital not found or mapping error for '%s' for user %s", loginReq.Hospital, loginReq.Username)
		if errors.Is(err, gorm.ErrRecordNotFound) { // Assuming GetHospitalIDByName returns this for not found
			return "", nil, errors.New("invalid hospital specified")
		}
		return "", nil, errors.New("error verifying hospital") // Generic internal error
	}

	if staff.HospitalID != inputHospitalID {
		log.Printf("Authentication failed: Hospital mismatch for user %s. Expected %d (%s), got %d (%s)",
			loginReq.Username, staff.HospitalID, staff.HospitalName, inputHospitalID, loginReq.Hospital)
		return "", nil, errors.New("invalid hospital for this user")
	}

	// 3. Verify the password
	if !utils.CheckPasswordHash(loginReq.Password, staff.PasswordHash) {
		log.Printf("Authentication failed: Invalid password for user %s", loginReq.Username)
		return "", nil, errors.New("invalid username or password") // Keep error message generic
	}

	// 4. Generate JWT Token
	// Use the jwtExpiry stored during InitializeAuthService
	expirationTime := time.Now().Add(jwtExpiry)
	claims := &Claims{
		UserID:     staff.ID,
		Username:   staff.Username,
		HospitalID: staff.HospitalID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", staff.ID), // Subject is typically the user ID
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		log.Printf("Error generating JWT token for user %s: %v", loginReq.Username, err)
		return "", nil, fmt.Errorf("could not generate token: %w", err)
	}

	log.Printf("Authentication successful for user: %s (Hospital ID: %d)", staff.Username, staff.HospitalID)
	staff.PasswordHash = "" // Don't return password hash
	return tokenString, staff, nil
}

// ValidateToken parses and validates a JWT token string.
func ValidateToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Return the secret key stored during initialization
		return jwtKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			log.Println("Token validation failed: Token is expired")
			return nil, errors.New("token is expired")
		}
		log.Printf("Token validation failed: %v", err)
		return nil, errors.New("invalid token")
	}

	if !token.Valid {
		log.Println("Token validation failed: Token is invalid (post-parsing check)")
		return nil, errors.New("invalid token")
	}

	log.Printf("Token validated successfully for user: %s (ID: %d, Hospital ID: %d)", claims.Username, claims.UserID, claims.HospitalID)
	return claims, nil
}
