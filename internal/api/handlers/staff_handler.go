package handlers

import (
	"errors"
	"hospital-middleware/internal/database"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/services"
	"hospital-middleware/pkg/utils"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateStaffHandler handles the creation of a new staff member.
func CreateStaffHandler(c *gin.Context) {
	var req models.StaffCreateRequest

	// Bind JSON request body to the struct
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON for staff creation: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Check if username already exists
	_, err := database.FindStaffByUsername(req.Username)
	if err == nil {
		// User found, username already exists
		log.Printf("Attempt to create staff with existing username: %s", req.Username)
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Other database error occurred
		log.Printf("Database error checking username %s: %v", req.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking username"})
		return
	}

	// Hash the password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		log.Printf("Error hashing password for user %s: %v", req.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	// Get Hospital ID from name
	hospitalID, err := database.GetHospitalIDByName(req.Hospital)
	if err != nil {
		log.Printf("Error finding hospital ID for name '%s': %v", req.Hospital, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hospital specified: " + err.Error()})
		return
	}

	// Create the staff model
	newStaff := &models.Staff{
		Username:     req.Username,
		PasswordHash: hashedPassword,
		HospitalID:   hospitalID,
		HospitalName: req.Hospital,
	}

	// Save to database
	if err := database.CreateStaff(newStaff); err != nil {
		log.Printf("Error creating staff %s in database: %v", req.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create staff member"})
		return
	}

	log.Printf("Successfully created staff: %s (Hospital: %s, ID: %d)", newStaff.Username, newStaff.HospitalName, newStaff.ID)

	// Return success response (don't return password hash)
	newStaff.PasswordHash = "" // Clear hash before returning
	c.JSON(http.StatusCreated, newStaff)
}

// LoginStaffHandler handles staff login attempts.
func LoginStaffHandler(c *gin.Context) {
	var req models.StaffLoginRequest

	// Bind JSON request body
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON for staff login: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Authenticate and generate token
	token, staff, err := services.AuthenticateStaff(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()}) // Ex. "invalid username or password", "invalid hospital"
		return
	}

	// Return token and basic staff info
	response := models.StaffLoginResponse{
		Token: token,
		Staff: *staff, // Dereference pointer, password hash is already cleared in AuthenticateStaff
	}
	c.JSON(http.StatusOK, response)
}
