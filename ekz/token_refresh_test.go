package ekz

import (
	"testing"

	"gopkg.in/h2non/gock.v1"
)

func TestTokenRefreshOn401(t *testing.T) {
	defer gock.Off()

	// Mock the initial 401 response for profile request
	gock.New("https://be.emob.ekz.ch").
		Get("/users/profile").
		MatchHeader("Authorization", "Token old-invalid-token").
		Reply(401).
		JSON(map[string]interface{}{
			"status_code": 401,
			"message":     "Unauthorized",
		})

	// Mock the login request that will happen during token refresh
	gock.New("https://be.emob.ekz.ch").
		Post("/users/log-in").
		JSON(loginRequest{
			Device:        "WEB",
			Email:         "test@example.com",
			Password:      "password123",
			IsSocialLogin: false,
		}).
		Reply(200).
		JSON(loginResponse{
			StatusCode: 200,
			Message:    "Login successful",
			Token:      "new-fresh-token",
			IsVerified: true,
		})

	// Mock the retry profile request with the new token
	gock.New("https://be.emob.ekz.ch").
		Get("/users/profile").
		MatchHeader("Authorization", "Token new-fresh-token").
		Reply(200).
		JSON(map[string]interface{}{
			"status_code": 200,
			"data": Profile{
				Personal: struct {
					FirstName                string      `json:"first_name"`
					LastName                 string      `json:"last_name"`
					Email                    string      `json:"email"`
					Language                 string      `json:"language"`
					LanguageOfCorrespondence string      `json:"language_of_correspondence"`
					MobilePhone              interface{} `json:"mobile_phone"`
					Street                   string      `json:"street"`
					HouseNumber              string      `json:"house_number"`
					ZipCode                  string      `json:"zip_code"`
					City                     string      `json:"city"`
					Country                  string      `json:"country"`
					CompanyName              string      `json:"company_name"`
					IsSocialLogin            bool        `json:"is_social_login"`
					Telefone                 string      `json:"telefone"`
					UserId                   int         `json:"user_id"`
				}{
					Email:   "test@example.com",
					UserId:  123,
					Country: "CH",
				},
			},
		})

	// Create client with invalid token
	config := &Config{
		Username: "test@example.com",
		Password: "password123",
		Token:    "old-invalid-token",
	}

	client, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Manually set the token to simulate having an old token
	client.setToken("old-invalid-token")

	// Make a request that should trigger 401 and token refresh
	profile, err := client.GetProfile()
	if err != nil {
		t.Fatalf("Expected profile request to succeed after token refresh, got error: %v", err)
	}

	if profile == nil {
		t.Fatal("Expected profile to be returned, got nil")
	}

	if profile.Personal.Email != "test@example.com" {
		t.Errorf("Expected email to be test@example.com, got %s", profile.Personal.Email)
	}

	// Verify that the token was updated
	newToken := client.getToken()
	if newToken != "new-fresh-token" {
		t.Errorf("Expected token to be updated to 'new-fresh-token', got '%s'", newToken)
	}

	// Verify all expected HTTP calls were made
	if !gock.IsDone() {
		t.Error("Not all expected HTTP mocks were called")
		for _, mock := range gock.Pending() {
			t.Logf("Pending mock: %s %s", mock.Request().Method, mock.Request().URLStruct.String())
		}
	}
}

func TestTokenRefreshFailsGracefully(t *testing.T) {
	defer gock.Off()

	// Mock the initial 401 response
	gock.New("https://be.emob.ekz.ch").
		Get("/users/profile").
		MatchHeader("Authorization", "Token old-invalid-token").
		Reply(401).
		JSON(map[string]interface{}{
			"status_code": 401,
			"message":     "Unauthorized",
		})

	// Mock login failure
	gock.New("https://be.emob.ekz.ch").
		Post("/users/log-in").
		Reply(401).
		JSON(map[string]interface{}{
			"status_code": 401,
			"message":     "Invalid credentials",
		})

	// Create client with invalid token and wrong credentials
	config := &Config{
		Username: "test@example.com",
		Password: "wrong-password",
		Token:    "old-invalid-token",
	}

	client, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	client.setToken("old-invalid-token")

	// Make a request that should trigger 401, attempt refresh, but fail
	_, err = client.GetProfile()
	if err == nil {
		t.Fatal("Expected error when token refresh fails")
	}

	// The error should be related to the profile request, not the login
	// (since we return the original 401 when refresh fails)
	if err.Error() != "unexpected status 401 Unauthorized" {
		t.Errorf("Expected '401 Unauthorized' error, got: %v", err)
	}
}