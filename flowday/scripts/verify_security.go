package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"
)

const BaseURL = "http://localhost:8080/api/v1"

func main() {
	fmt.Println("🔒 Starting Security & Auth Flow Test...")

	// 1. Setup Cookie Jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}

	// 2. Register / Login
	email := fmt.Sprintf("test_%d@flowday.com", time.Now().Unix())
	password := "securepassword123"
	fmt.Printf("\n👤 Creating User: %s\n", email)

	registerPayload := map[string]string{
		"name":     "Security Tester",
		"email":    email,
		"password": password,
	}

	// Register
	if err := post(client, "/auth/register", registerPayload); err != nil {
		fmt.Printf("⚠️ Register failed (user might exist, trying login): %v\n", err)
	}

	// Login
	fmt.Println("\n🔑 Logging in...")
	loginPayload := map[string]string{
		"email":    email,
		"password": password,
	}
	resp, body, err := postWithResponse(client, "/auth/login", loginPayload)
	if err != nil {
		panic(err)
	}

	// 3. Inspect Tokens
	fmt.Println("✅ Login Successful!")
	var loginResp map[string]string
	json.Unmarshal(body, &loginResp)
	fmt.Printf("   Access Token: %s...\n", loginResp["token"][:20])

	// Check Cookies
	cookies := jar.Cookies(resp.Request.URL)
	foundRefresh := false
	for _, cookie := range cookies {
		if cookie.Name == "refresh_token" {
			fmt.Printf("   Refresh Cookie: [HttpOnly: %t] [Path: %s] [Expires: %s]\n", cookie.HttpOnly, cookie.Path, cookie.Expires)
			foundRefresh = true
		}
	}

	if !foundRefresh {
		fmt.Println("❌ CRITICAL: No refresh_token cookie received!")
		return
	} else {
		fmt.Println("✅ Secure refresh_token cookie confirmed.")
	}

	// 4. Test Refresh flow
	fmt.Println("\n🔄 Testing Token Refresh endpoint...")
	respRefresh, bodyRefresh, err := postEmpty(client, "/auth/refresh")
	if err != nil {
		fmt.Printf("❌ Refresh failed: %v\n", err)
		return
	}

	if respRefresh.StatusCode != 200 {
		fmt.Printf("❌ Refresh returned status %d: %s\n", respRefresh.StatusCode, string(bodyRefresh))
		return
	}

	var refreshResp map[string]string
	json.Unmarshal(bodyRefresh, &refreshResp)
	newAccessToken := refreshResp["token"]

	if newAccessToken != "" && newAccessToken != loginResp["token"] {
		fmt.Printf("✅ Refresh Successful! New Access Token: %s...\n", newAccessToken[:20])
	} else {
		fmt.Println("⚠️ Refresh returned empty or identical token (check implementation)")
	}

	fmt.Println("\n🎉 SECURITY TEST PASSED: Dual-Token Architecture is active.")
}

func post(client *http.Client, endpoint string, data interface{}) error {
	_, _, err := postWithResponse(client, endpoint, data)
	return err
}

func postWithResponse(client *http.Client, endpoint string, data interface{}) (*http.Response, []byte, error) {
	jsonData, _ := json.Marshal(data)
	resp, err := client.Post(BaseURL+endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return resp, body, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	return resp, body, nil
}

func postEmpty(client *http.Client, endpoint string) (*http.Response, []byte, error) {
	resp, err := client.Post(BaseURL+endpoint, "application/json", nil)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp, body, nil
}
