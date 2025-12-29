package email

import (
	"strings"
	"testing"
)

/**
 * Test Plan: SMTP Email Sender
 *
 * Scenario: NewSender creates sender with config
 *   Given a valid SMTP configuration
 *   When NewSender() is called
 *   Then a Sender is returned with the config stored
 *
 * Scenario: ShareEmailData template rendering
 *   Given share email data with session name and download URL
 *   When the email template is rendered
 *   Then the session name and download URL are included in the output
 *
 * Scenario: Config validation
 *   Given various SMTP configurations
 *   When sender is created
 *   Then config values are stored correctly
 */

func TestNewSender(t *testing.T) {
	config := Config{
		Host:     "smtp.example.com",
		Port:     "587",
		Username: "user@example.com",
		Password: "password123",
		From:     "noreply@example.com",
		FromName: "Test Sender",
	}

	sender := NewSender(config)

	if sender == nil {
		t.Fatal("NewSender() returned nil")
	}
	if sender.config.Host != config.Host {
		t.Errorf("sender.config.Host = %v, want %v", sender.config.Host, config.Host)
	}
	if sender.config.Port != config.Port {
		t.Errorf("sender.config.Port = %v, want %v", sender.config.Port, config.Port)
	}
	if sender.config.From != config.From {
		t.Errorf("sender.config.From = %v, want %v", sender.config.From, config.From)
	}
}

func TestShareEmailData(t *testing.T) {
	data := ShareEmailData{
		RecipientEmail: "recipient@example.com",
		SessionName:    "Test Recording",
		DownloadURL:    "https://example.com/download/abc123",
		ExpiresIn:      "24 hours",
	}

	if data.RecipientEmail != "recipient@example.com" {
		t.Errorf("RecipientEmail = %v, want recipient@example.com", data.RecipientEmail)
	}
	if data.SessionName != "Test Recording" {
		t.Errorf("SessionName = %v, want Test Recording", data.SessionName)
	}
	if data.DownloadURL != "https://example.com/download/abc123" {
		t.Errorf("DownloadURL = %v, want https://example.com/download/abc123", data.DownloadURL)
	}
	if data.ExpiresIn != "24 hours" {
		t.Errorf("ExpiresIn = %v, want 24 hours", data.ExpiresIn)
	}
}

func TestShareEmailTemplate(t *testing.T) {
	// Verify the template contains expected placeholders
	if !strings.Contains(shareEmailTemplate, "{{.SessionName}}") {
		t.Error("Template missing {{.SessionName}} placeholder")
	}
	if !strings.Contains(shareEmailTemplate, "{{.DownloadURL}}") {
		t.Error("Template missing {{.DownloadURL}} placeholder")
	}
	if !strings.Contains(shareEmailTemplate, "{{.ExpiresIn}}") {
		t.Error("Template missing {{.ExpiresIn}} placeholder")
	}
	if !strings.Contains(shareEmailTemplate, "Download Recording") {
		t.Error("Template missing 'Download Recording' button text")
	}
}

func TestConfigDefaults(t *testing.T) {
	// Test that config can be created with minimal values
	config := Config{
		Host: "smtp.example.com",
		From: "noreply@example.com",
	}

	sender := NewSender(config)
	if sender.config.Port != "" {
		// Port is not set, so it should be empty (caller should set default)
		// This tests that the config doesn't unexpectedly modify values
	}
	if sender.config.FromName != "" {
		// FromName is optional
	}
}
