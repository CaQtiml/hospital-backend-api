package test

import (
	"bytes"
	"encoding/json"
	"fmt" // Import fmt for generating unique usernames
	"hospital-middleware/internal/api"
	"hospital-middleware/internal/config"
	"hospital-middleware/internal/database"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/services"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time" // Import time for unique usernames

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm" // Import gorm
)

var testRouter *gin.Engine

// var testToken string // Store token for authenticated tests
var testDB *gorm.DB // Make DB instance accessible

// Helper function to generate unique usernames for tests
func uniqueUsername(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// Setup runs before all tests in the package
func TestMain(m *testing.M) {
	log.Println("Setting up test environment...")

	// Load .env file from the parent directory (project root)
	// Adjust path if your test file is nested deeper
	err := godotenv.Load("../.env")
	if err != nil {
		// Try loading from current directory if ../.env fails (e.g., running tests from root)
		err = godotenv.Load(".env")
		if err != nil {
			log.Printf("Warning: Could not load .env file: %v. Relying on environment variables.", err)
		} else {
			log.Println("Test environment variables loaded from ./ .env")
		}
	} else {
		log.Println("Test environment variables loaded from ../.env")
	}

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Load test configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config for testing: %v", err)
	}
	log.Printf("Test Config Loaded: DB_HOST=%s, DB_PORT=%s, DB_NAME=%s, DB_USER=%s", cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser)

	// Connect to the test database
	if err := database.Connect(cfg); err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}
	testDB = database.GetDB() // Store DB instance

	// --- Optional: Clean database before starting test suite ---
	// log.Println("Cleaning test database before suite...")
	// CleanTestData(testDB) // Implement or uncomment this if needed
	// ---------------------------------------------------------

	// Initialize services
	services.InitializeAuthService(cfg)

	// Setup router
	testRouter = api.SetupRouter()

	// Run tests
	exitCode := m.Run()

	// --- Optional: Clean database after test suite ---
	// log.Println("Cleaning test database after suite...")
	// CleanTestData(testDB) // Implement or uncomment this if needed
	// ----------------------------------------------------

	// Teardown (optional)
	log.Println("Tearing down test environment...")
	// Close DB connection if necessary (GORM might handle pooling)

	os.Exit(exitCode)
}

// Helper to perform requests
func performRequest(router *gin.Engine, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var req *http.Request
	var err error

	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req, err = http.NewRequest(method, path, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, path, nil)
	}
	if err != nil {
		panic(err) // Should not happen in tests
	}

	// Add Authorization header if token is provided
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// --- Test Cases ---

func TestHealthCheck(t *testing.T) {
	rr := performRequest(testRouter, "GET", "/health", nil, "")

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"status":"UP"`)
}

func TestCreateStaffHandler_Success(t *testing.T) {
	username := uniqueUsername("testuser_create") // Use unique username
	staffData := models.StaffCreateRequest{
		Username: username,
		Password: "password123",
		Hospital: "Hospital A", // Make sure this hospital exists via GetHospitalIDByName mock/setup
	}

	// Register cleanup function *before* creating the data
	t.Cleanup(func() {
		log.Printf("Cleaning up staff: %s", username)
		// Use Unscoped() to permanently delete if using soft deletes, otherwise simple Delete is fine
		err := testDB.Unscoped().Where("username = ?", username).Delete(&models.Staff{}).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			log.Printf("Error cleaning up staff %s: %v", username, err)
		}
	})

	rr := performRequest(testRouter, "POST", "/api/v1/staff/create", staffData, "")

	assert.Equal(t, http.StatusCreated, rr.Code)

	var createdStaff models.Staff
	err := json.Unmarshal(rr.Body.Bytes(), &createdStaff)
	assert.NoError(t, err)
	assert.Equal(t, staffData.Username, createdStaff.Username)
	assert.Equal(t, staffData.Hospital, createdStaff.HospitalName)
	assert.NotEmpty(t, createdStaff.ID)
	assert.Empty(t, createdStaff.PasswordHash) // Ensure password hash is not returned
}

func TestCreateStaffHandler_DuplicateUsername(t *testing.T) {
	// 1. Create a user first
	username := uniqueUsername("testuser_duplicate") // Use unique username
	staffData := models.StaffCreateRequest{
		Username: username,
		Password: "password123",
		Hospital: "Hospital A",
	}

	// Cleanup for the initially created user
	t.Cleanup(func() {
		log.Printf("Cleaning up staff: %s", username)
		err := testDB.Unscoped().Where("username = ?", username).Delete(&models.Staff{}).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			log.Printf("Error cleaning up staff %s: %v", username, err)
		}
	})

	// Perform the initial creation
	rrCreate := performRequest(testRouter, "POST", "/api/v1/staff/create", staffData, "")
	assert.Equal(t, http.StatusCreated, rrCreate.Code, "Initial user creation failed")
	if rrCreate.Code != http.StatusCreated {
		t.FailNow() // Stop test if setup fails
	}

	// 2. Attempt to create again
	rrDuplicate := performRequest(testRouter, "POST", "/api/v1/staff/create", staffData, "")

	assert.Equal(t, http.StatusConflict, rrDuplicate.Code)
	assert.Contains(t, rrDuplicate.Body.String(), "Username already exists")
}

func TestCreateStaffHandler_BadData(t *testing.T) {
	badStaffData := gin.H{ // Missing required fields
		"username": "testuser_bad",
		// Missing password and hospital
	}
	rr := performRequest(testRouter, "POST", "/api/v1/staff/create", badStaffData, "")

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid request body") // Or more specific binding error
}

func TestLoginStaffHandler_Success(t *testing.T) {
	// 1. Ensure the user exists (create if necessary)
	username := uniqueUsername("testuser_login") // Use unique username
	password := "password123"
	hospital := "Hospital B" // Use a different hospital for variety
	staffData := models.StaffCreateRequest{
		Username: username,
		Password: password,
		Hospital: hospital,
	}

	// Cleanup for the created user
	t.Cleanup(func() {
		log.Printf("Cleaning up staff: %s", username)
		err := testDB.Unscoped().Where("username = ?", username).Delete(&models.Staff{}).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			log.Printf("Error cleaning up staff %s: %v", username, err)
		}
	})

	// Create the user for login test
	rrCreate := performRequest(testRouter, "POST", "/api/v1/staff/create", staffData, "")
	assert.Equal(t, http.StatusCreated, rrCreate.Code, "Setup: User creation for login test failed")
	if rrCreate.Code != http.StatusCreated {
		t.FailNow()
	}

	// 2. Attempt login
	loginData := models.StaffLoginRequest{
		Username: username,
		Password: password,
		Hospital: hospital,
	}
	rrLogin := performRequest(testRouter, "POST", "/api/v1/staff/login", loginData, "")

	assert.Equal(t, http.StatusOK, rrLogin.Code)

	var loginResponse models.StaffLoginResponse
	err := json.Unmarshal(rrLogin.Body.Bytes(), &loginResponse)
	assert.NoError(t, err)
	assert.NotEmpty(t, loginResponse.Token, "Token should not be empty on successful login")
	assert.Equal(t, username, loginResponse.Staff.Username)
	assert.Equal(t, hospital, loginResponse.Staff.HospitalName)
	assert.Empty(t, loginResponse.Staff.PasswordHash)

	// Store the token for authenticated tests (if needed by subsequent tests in sequence, less ideal for parallel)
	// Consider getting a fresh token within tests that need it for better isolation.
	// testToken = loginResponse.Token
	// log.Printf("Obtained test token: %s...", testToken[:10])
}

func TestLoginStaffHandler_WrongPassword(t *testing.T) {
	// 1. Create user
	username := uniqueUsername("testuser_wrongpass")
	password := "correctpassword"
	hospital := "Hospital A"
	staffData := models.StaffCreateRequest{Username: username, Password: password, Hospital: hospital}

	t.Cleanup(func() {
		log.Printf("Cleaning up staff: %s", username)
		err := testDB.Unscoped().Where("username = ?", username).Delete(&models.Staff{}).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			log.Printf("Error cleaning up staff %s: %v", username, err)
		}
	})

	rrCreate := performRequest(testRouter, "POST", "/api/v1/staff/create", staffData, "")
	assert.Equal(t, http.StatusCreated, rrCreate.Code, "Setup: User creation for wrong password test failed")
	if rrCreate.Code != http.StatusCreated {
		t.FailNow()
	}

	// 2. Attempt login with wrong password
	loginData := models.StaffLoginRequest{
		Username: username,
		Password: "wrongpassword",
		Hospital: hospital,
	}
	rrLogin := performRequest(testRouter, "POST", "/api/v1/staff/login", loginData, "")

	assert.Equal(t, http.StatusUnauthorized, rrLogin.Code)
	assert.Contains(t, rrLogin.Body.String(), "invalid username or password")
}

func TestLoginStaffHandler_WrongHospital(t *testing.T) {
	// 1. Create user
	username := uniqueUsername("testuser_wronghosp")
	password := "password123"
	correctHospital := "Hospital B"
	wrongHospital := "Hospital C" // Non-existent or wrong hospital
	staffData := models.StaffCreateRequest{Username: username, Password: password, Hospital: correctHospital}

	t.Cleanup(func() {
		log.Printf("Cleaning up staff: %s", username)
		err := testDB.Unscoped().Where("username = ?", username).Delete(&models.Staff{}).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			log.Printf("Error cleaning up staff %s: %v", username, err)
		}
	})

	rrCreate := performRequest(testRouter, "POST", "/api/v1/staff/create", staffData, "")
	assert.Equal(t, http.StatusCreated, rrCreate.Code, "Setup: User creation for wrong hospital test failed")
	if rrCreate.Code != http.StatusCreated {
		t.FailNow()
	}

	// 2. Attempt login with wrong hospital
	loginData := models.StaffLoginRequest{
		Username: username,
		Password: password,
		Hospital: wrongHospital,
	}
	rrLogin := performRequest(testRouter, "POST", "/api/v1/staff/login", loginData, "")

	assert.Equal(t, http.StatusUnauthorized, rrLogin.Code)
	// The error might be "invalid hospital specified" or "invalid hospital for this user" depending on implementation
	assert.Contains(t, rrLogin.Body.String(), "error verifying hospital")
}

// --- Auth Required Tests ---

// Helper to get a valid token for authenticated tests
func getAuthToken(t *testing.T, username, password, hospital string) string {
	// Ensure user exists
	staffData := models.StaffCreateRequest{Username: username, Password: password, Hospital: hospital}
	rrCreate := performRequest(testRouter, "POST", "/api/v1/staff/create", staffData, "")
	// Handle potential conflict if user already exists from a previous failed cleanup, maybe try login directly?
	// For simplicity, we assume create works or the test setup ensures a clean slate or unique usernames work.
	if rrCreate.Code != http.StatusCreated && rrCreate.Code != http.StatusConflict {
		log.Printf("Failed to ensure user %s exists for token generation: Status %d", username, rrCreate.Code)
		t.Fatalf("Setup failed: Could not create/ensure user %s exists for token generation", username)
	}
	if rrCreate.Code == http.StatusCreated {
		// Need to clean up this user if we created it here
		t.Cleanup(func() {
			log.Printf("Cleaning up helper staff: %s", username)
			err := testDB.Unscoped().Where("username = ?", username).Delete(&models.Staff{}).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				log.Printf("Error cleaning up helper staff %s: %v", username, err)
			}
		})
	}

	// Attempt login
	loginData := models.StaffLoginRequest{Username: username, Password: password, Hospital: hospital}
	rrLogin := performRequest(testRouter, "POST", "/api/v1/staff/login", loginData, "")
	assert.Equal(t, http.StatusOK, rrLogin.Code, "Helper: Login failed during token generation")
	if rrLogin.Code != http.StatusOK {
		t.Fatalf("Setup failed: Could not log in user %s for token generation", username)
	}

	var loginResponse models.StaffLoginResponse
	err := json.Unmarshal(rrLogin.Body.Bytes(), &loginResponse)
	assert.NoError(t, err)
	assert.NotEmpty(t, loginResponse.Token)
	return loginResponse.Token
}

func TestSearchPatientHandler_Unauthorized(t *testing.T) {
	rr := performRequest(testRouter, "GET", "/api/v1/patient/search?first_name_en=Test", nil, "") // No token

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "Authorization header required")
}

func TestSearchPatientHandler_Authorized_NoResults(t *testing.T) {
	// Get a token for a user (creating the user and cleaning up automatically)
	tokenUsername := uniqueUsername("testuser_search_auth")
	authToken := getAuthToken(t, tokenUsername, "password123", "Hospital A")
	assert.NotEmpty(t, authToken)

	// Search for a patient unlikely to exist
	// Use query parameters matching the PatientSearchQuery struct fields (e.g., first_name_en)
	rr := performRequest(testRouter, "GET", "/api/v1/patient/search?first_name_en=NoSuchPatientXYZ123", nil, authToken)

	assert.Equal(t, http.StatusOK, rr.Code) // Should return 200 OK with empty list
	bodyString := rr.Body.String()
	assert.True(t, bodyString == "[]" || bodyString == "null", "Expected empty array '[]' or 'null', got: %s", bodyString) // Allow [] or null
}

// --- Helper for Cleaning ---

// CleanTestData completely wipes relevant tables. Use with caution.
// Prefer per-test cleanup (t.Cleanup) for better isolation.
func CleanTestData(db *gorm.DB) {
	log.Println("WARNING: Clearing ALL staff and patient data from test DB!")
	// Order matters due to potential foreign key constraints if they were enforced
	err := db.Exec("TRUNCATE TABLE patients RESTART IDENTITY CASCADE").Error // Use TRUNCATE for speed, CASCADE if FKs exist
	if err != nil {
		log.Printf("Error truncating patients: %v", err)
	}
	err = db.Exec("TRUNCATE TABLE staff RESTART IDENTITY CASCADE").Error
	if err != nil {
		log.Printf("Error truncating staff: %v", err)
	}
	log.Println("Test data cleared.")
}
