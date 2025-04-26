package api

import (
	"hospital-middleware/internal/api/handlers"
	"hospital-middleware/internal/api/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRouter configures the Gin router with all application routes.
func SetupRouter() *gin.Engine {
	// gin.SetMode(gin.ReleaseMode) // Uncomment for production
	router := gin.Default()

	// Health Check Endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	apiV1 := router.Group("/api/v1")
	{
		staffGroup := apiV1.Group("/staff")
		{
			staffGroup.POST("/create", handlers.CreateStaffHandler)
			staffGroup.POST("/login", handlers.LoginStaffHandler)
		}

		patientGroup := apiV1.Group("/patient")
		{
			// Apply authentication middleware ONLY to routes that require login
			patientGroup.Use(middleware.AuthRequired()) // Apply to all routes within this group
			patientGroup.GET("/search", handlers.SearchPatientHandler)
		}
	}

	// Handle 404 Not Found routes
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
	})

	return router
}
