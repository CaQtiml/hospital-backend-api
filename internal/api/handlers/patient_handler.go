package handlers

import (
	"hospital-middleware/internal/api/middleware"
	"hospital-middleware/internal/database"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/services"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SearchPatientHandler handles searching for patients. Requires authentication.
func SearchPatientHandler(c *gin.Context) {
	// 1. Get Claims from context (set by AuthRequired middleware)
	claimsInterface, exists := c.Get(middleware.ContextKeyClaims)
	if !exists {
		log.Println("Error in SearchPatientHandler: Claims not found in context. Middleware might be missing.")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required (claims not found)"})
		return
	}

	claims, ok := claimsInterface.(*services.Claims)
	if !ok {
		log.Println("Error in SearchPatientHandler: Could not assert claims type.")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error processing authentication"})
		return
	}

	// Staff's hospital ID is now available in claims.HospitalID
	staffHospitalID := claims.HospitalID
	log.Printf("Patient search initiated by staff %s (Hospital ID: %d)", claims.Username, staffHospitalID)

	// 2. Bind Query Parameters
	var searchQuery models.PatientSearchQuery
	if err := c.ShouldBindQuery(&searchQuery); err != nil {
		log.Printf("Error binding query parameters for patient search: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
		return
	}

	// Log the received search query
	log.Printf("Search query parameters: %+v", searchQuery)

	// 3. Perform Search using Database function
	// Pass the search criteria and the staff's hospital ID for filtering
	patients, err := database.SearchPatients(&searchQuery, staffHospitalID)
	if err != nil {
		log.Printf("Error searching patients in database for hospital %d: %v", staffHospitalID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error during patient search"})
		return
	}

	// 4. Return Results
	log.Printf("Found %d patients matching criteria for hospital %d", len(patients), staffHospitalID)
	if len(patients) == 0 {
		// Return empty list, not an error, if no patients match
		c.JSON(http.StatusOK, []models.Patient{})
		return
	}

	c.JSON(http.StatusOK, patients)
}
