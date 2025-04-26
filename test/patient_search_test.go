// hospital/test/patient_search_test.go
package test

import (
	"encoding/json"
	"fmt"
	"hospital-middleware/internal/models"
	"log"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// --- Patient Seeding Helpers ---

// Helper to seed a patient and ensure cleanup for idempotency
func seedPatient(t *testing.T, patient *models.Patient) {
	// Ensure required fields are set to avoid DB errors if constraints exist
	if patient.HospitalID == 0 {
		patient.HospitalID = 1 // Default if not set, adjust if needed based on GetHospitalIDByName
	}
	if patient.PatientHN == "" {
		// Generate a unique HN if needed, simplified here
		patient.PatientHN = fmt.Sprintf("HN_TEST_%d", time.Now().UnixNano())
	}

	// Use the global testDB variable defined in api_test.go (or main_test.go)
	err := testDB.Create(patient).Error
	if err != nil {
		t.Fatalf("Failed to seed patient %+v: %v", *patient, err)
	}

	t.Cleanup(func() {
		log.Printf("Cleaning up patient ID: %d, HN: %s", patient.ID, patient.PatientHN)
		// Use Unscoped() if using soft deletes
		// Delete based on primary key for safety
		err := testDB.Unscoped().Delete(&models.Patient{}, patient.ID).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			log.Printf("Error cleaning up patient ID %d: %v", patient.ID, err)
		}
	})
}

// Base patient data for tests
func createTestPatient(hospitalID uint) *models.Patient {
	dobStr := "1990-05-15"
	dob, _ := time.Parse("2006-01-02", dobStr)
	return &models.Patient{
		HospitalID:  hospitalID, // Set per test case
		PatientHN:   fmt.Sprintf("TESTHN%d", time.Now().UnixNano()),
		FirstNameTH: "ทดสอบ",
		LastNameTH:  "นามสกุล",
		FirstNameEN: "Test",
		LastNameEN:  "Patient",
		DateOfBirth: &dob,
		NationalID:  fmt.Sprintf("NID%d", time.Now().UnixNano()),
		PassportID:  fmt.Sprintf("PASS%d", time.Now().UnixNano()),
		PhoneNumber: fmt.Sprintf("08%d", time.Now().UnixNano()%100000000),
		Email:       fmt.Sprintf("test.patient%d@example.com", time.Now().UnixNano()),
		Gender:      "M",
	}
}

// --- Patient Search Test Cases ---

func TestSearchPatientHandler_FoundByNationalID(t *testing.T) {
	// 1. Seed Patient Data for Hospital A (ID 1)
	testPatient := createTestPatient(1)
	testPatient.NationalID = "NID1234567890"
	seedPatient(t, testPatient)

	// 2. Get Token for Staff from Hospital A
	tokenUsername := uniqueUsername("staff_hospA_nid")
	authToken := getAuthToken(t, tokenUsername, "password123", "Hospital A") // Uses helper from api_test.go
	assert.NotEmpty(t, authToken)

	// 3. Perform Search
	query := url.Values{}
	query.Add("national_id", testPatient.NationalID)
	searchURL := "/api/v1/patient/search?" + query.Encode()

	rr := performRequest(testRouter, "GET", searchURL, nil, authToken) // Uses helper from api_test.go
	assert.Equal(t, http.StatusOK, rr.Code)

	// 4. Assertions
	var results []models.Patient
	err := json.Unmarshal(rr.Body.Bytes(), &results)
	assert.NoError(t, err)
	assert.Len(t, results, 1, "Expected exactly one patient result")
	if len(results) == 1 {
		assert.Equal(t, testPatient.NationalID, results[0].NationalID)
		assert.Equal(t, testPatient.FirstNameEN, results[0].FirstNameEN)
		assert.Equal(t, testPatient.HospitalID, results[0].HospitalID) // Verify correct hospital
	}
}

func TestSearchPatientHandler_FoundByPassportID(t *testing.T) {
	// 1. Seed Patient Data for Hospital B (ID 2)
	testPatient := createTestPatient(2)
	testPatient.PassportID = "PASSXYZ987"
	seedPatient(t, testPatient)

	// 2. Get Token for Staff from Hospital B
	tokenUsername := uniqueUsername("staff_hospB_pass")
	authToken := getAuthToken(t, tokenUsername, "password123", "Hospital B")
	assert.NotEmpty(t, authToken)

	// 3. Perform Search
	query := url.Values{}
	query.Add("passport_id", testPatient.PassportID)
	searchURL := "/api/v1/patient/search?" + query.Encode()

	rr := performRequest(testRouter, "GET", searchURL, nil, authToken)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 4. Assertions
	var results []models.Patient
	err := json.Unmarshal(rr.Body.Bytes(), &results)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	if len(results) == 1 {
		assert.Equal(t, testPatient.PassportID, results[0].PassportID)
		assert.Equal(t, testPatient.FirstNameEN, results[0].FirstNameEN)
		assert.Equal(t, testPatient.HospitalID, results[0].HospitalID) // Verify correct hospital
	}
}

func TestSearchPatientHandler_FoundByNameTH(t *testing.T) {
	// 1. Seed Patient Data (Hospital A)
	testPatient := createTestPatient(1)
	testPatient.FirstNameTH = "สมหมายไท"
	testPatient.LastNameTH = "ทดสอบไท"
	seedPatient(t, testPatient)

	// 2. Get Token (Hospital A)
	tokenUsername := uniqueUsername("staff_hospA_nameth")
	authToken := getAuthToken(t, tokenUsername, "password123", "Hospital A")
	assert.NotEmpty(t, authToken)

	// 3. Perform Search (Partial First Name)
	query := url.Values{}
	query.Add("first_name_th", "สมหมาย") // Partial match
	searchURL := "/api/v1/patient/search?" + query.Encode()

	rr := performRequest(testRouter, "GET", searchURL, nil, authToken)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 4. Assertions
	var results []models.Patient
	err := json.Unmarshal(rr.Body.Bytes(), &results)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1, "Expected at least one result for partial name match")
	found := false
	for _, p := range results {
		if p.ID == testPatient.ID { // Check if our seeded patient is among results
			assert.Equal(t, testPatient.FirstNameTH, p.FirstNameTH)
			assert.Equal(t, testPatient.HospitalID, p.HospitalID)
			found = true
			break
		}
	}
	assert.True(t, found, "Seeded patient not found in results for partial name search")
}

func TestSearchPatientHandler_FoundByNameEN(t *testing.T) {
	// 1. Seed Patient Data (Hospital B)
	testPatient := createTestPatient(2)
	testPatient.FirstNameEN = "SpecificName"
	testPatient.LastNameEN = "PatientEN"
	seedPatient(t, testPatient)

	// 2. Get Token (Hospital B)
	tokenUsername := uniqueUsername("staff_hospB_nameen")
	authToken := getAuthToken(t, tokenUsername, "password123", "Hospital B")
	assert.NotEmpty(t, authToken)

	// 3. Perform Search
	query := url.Values{}
	query.Add("last_name_en", "Patient")
	searchURL := "/api/v1/patient/search?" + query.Encode()

	rr := performRequest(testRouter, "GET", searchURL, nil, authToken)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 4. Assertions
	var results []models.Patient
	err := json.Unmarshal(rr.Body.Bytes(), &results)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
	found := false
	for _, p := range results {
		if p.ID == testPatient.ID {
			assert.Equal(t, testPatient.LastNameEN, p.LastNameEN)
			assert.Equal(t, testPatient.HospitalID, p.HospitalID)
			found = true
			break
		}
	}
	assert.True(t, found, "Seeded patient not found in results for partial name search")
}

func TestSearchPatientHandler_FoundByDOB(t *testing.T) {
	// 1. Seed Patient Data (Hospital A)
	dobStr := "1985-11-20"
	dob, _ := time.Parse("2006-01-02", dobStr)
	testPatient := createTestPatient(1)
	testPatient.DateOfBirth = &dob
	seedPatient(t, testPatient)

	// 2. Get Token (Hospital A)
	tokenUsername := uniqueUsername("staff_hospA_dob")
	authToken := getAuthToken(t, tokenUsername, "password123", "Hospital A")
	assert.NotEmpty(t, authToken)

	// 3. Perform Search
	query := url.Values{}
	query.Add("date_of_birth", dobStr) // Use YYYY-MM-DD format
	searchURL := "/api/v1/patient/search?" + query.Encode()

	rr := performRequest(testRouter, "GET", searchURL, nil, authToken)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 4. Assertions
	var results []models.Patient
	err := json.Unmarshal(rr.Body.Bytes(), &results)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1) // DOB might not be unique
	found := false
	for _, p := range results {
		if p.ID == testPatient.ID {
			assert.NotNil(t, p.DateOfBirth)
			assert.Equal(t, dobStr, p.DateOfBirth.Format("2006-01-02"))
			assert.Equal(t, testPatient.HospitalID, p.HospitalID)
			found = true
			break
		}
	}
	assert.True(t, found, "Seeded patient not found in results for DOB search")
}

func TestSearchPatientHandler_FoundByPhoneNumber(t *testing.T) {
	// 1. Seed Patient Data (Hospital B)
	testPatient := createTestPatient(2)
	testPatient.PhoneNumber = "0898765432"
	seedPatient(t, testPatient)

	// 2. Get Token (Hospital B)
	tokenUsername := uniqueUsername("staff_hospB_phone")
	authToken := getAuthToken(t, tokenUsername, "password123", "Hospital B")
	assert.NotEmpty(t, authToken)

	// 3. Perform Search
	query := url.Values{}
	query.Add("phone_number", testPatient.PhoneNumber)
	searchURL := "/api/v1/patient/search?" + query.Encode()

	rr := performRequest(testRouter, "GET", searchURL, nil, authToken)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 4. Assertions
	var results []models.Patient
	err := json.Unmarshal(rr.Body.Bytes(), &results)
	assert.NoError(t, err)
	assert.Len(t, results, 1) // Expect exact match for phone
	if len(results) == 1 {
		assert.Equal(t, testPatient.PhoneNumber, results[0].PhoneNumber)
		assert.Equal(t, testPatient.HospitalID, results[0].HospitalID)
	}
}

func TestSearchPatientHandler_FoundByEmail(t *testing.T) {
	// 1. Seed Patient Data (Hospital A)
	testPatient := createTestPatient(1)
	testPatient.Email = "specific.email.test@sample.org"
	seedPatient(t, testPatient)

	// 2. Get Token (Hospital A)
	tokenUsername := uniqueUsername("staff_hospA_email")
	authToken := getAuthToken(t, tokenUsername, "password123", "Hospital A")
	assert.NotEmpty(t, authToken)

	// 3. Perform Search
	query := url.Values{}
	query.Add("email", testPatient.Email)
	searchURL := "/api/v1/patient/search?" + query.Encode()

	rr := performRequest(testRouter, "GET", searchURL, nil, authToken)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 4. Assertions
	var results []models.Patient
	err := json.Unmarshal(rr.Body.Bytes(), &results)
	assert.NoError(t, err)
	assert.Len(t, results, 1) // Expect exact match for email
	if len(results) == 1 {
		assert.Equal(t, testPatient.Email, results[0].Email)
		assert.Equal(t, testPatient.HospitalID, results[0].HospitalID)
	}
}

func TestSearchPatientHandler_FoundByMultipleFields(t *testing.T) {
	// 1. Seed Patient Data (Hospital B)
	testPatient := createTestPatient(2)
	testPatient.FirstNameEN = "Multi"
	testPatient.LastNameEN = "Criteria"
	testPatient.NationalID = "NIDMULTI999"
	seedPatient(t, testPatient)

	// Seed another patient that *doesn't* match all criteria
	otherPatient := createTestPatient(2)
	otherPatient.FirstNameEN = "Multi"
	otherPatient.LastNameEN = "Mismatch"
	otherPatient.NationalID = "NIDOTHER000"
	seedPatient(t, otherPatient)

	// 2. Get Token (Hospital B)
	tokenUsername := uniqueUsername("staff_hospB_multi")
	authToken := getAuthToken(t, tokenUsername, "password123", "Hospital B")
	assert.NotEmpty(t, authToken)

	// 3. Perform Search using multiple fields
	query := url.Values{}
	query.Add("first_name_en", testPatient.FirstNameEN) // "Multi"
	query.Add("last_name_en", "Criter")                 // Partial match for LastNameEN
	query.Add("national_id", testPatient.NationalID)    // Exact match for NID
	searchURL := "/api/v1/patient/search?" + query.Encode()

	rr := performRequest(testRouter, "GET", searchURL, nil, authToken)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 4. Assertions
	var results []models.Patient
	err := json.Unmarshal(rr.Body.Bytes(), &results)
	assert.NoError(t, err)
	assert.Len(t, results, 1, "Expected only the one patient matching all criteria")
	if len(results) == 1 {
		assert.Equal(t, testPatient.ID, results[0].ID) // Check ID specifically
		assert.Equal(t, testPatient.FirstNameEN, results[0].FirstNameEN)
		assert.Equal(t, testPatient.LastNameEN, results[0].LastNameEN)
		assert.Equal(t, testPatient.NationalID, results[0].NationalID)
		assert.Equal(t, testPatient.HospitalID, results[0].HospitalID)
	}
}

func TestSearchPatientHandler_NotFoundWrongHospital(t *testing.T) {
	// 1. Seed Patient Data for Hospital A (ID 1)
	testPatient := createTestPatient(1)
	testPatient.NationalID = "NIDWRONGHOSP1"
	seedPatient(t, testPatient)

	// 2. Get Token for Staff from Hospital B (ID 2 - Different Hospital)
	tokenUsername := uniqueUsername("staff_hospB_wrong")
	authToken := getAuthToken(t, tokenUsername, "password123", "Hospital B")
	assert.NotEmpty(t, authToken)

	// 3. Perform Search using Hospital B staff token for Hospital A patient's NID
	query := url.Values{}
	query.Add("national_id", testPatient.NationalID)
	searchURL := "/api/v1/patient/search?" + query.Encode()

	rr := performRequest(testRouter, "GET", searchURL, nil, authToken)
	assert.Equal(t, http.StatusOK, rr.Code) // Should still be OK status

	// 4. Assertions - Expect empty results because staff is from wrong hospital
	var results []models.Patient
	err := json.Unmarshal(rr.Body.Bytes(), &results)
	assert.NoError(t, err)
	assert.Len(t, results, 0, "Expected zero results when searching from wrong hospital")
}
