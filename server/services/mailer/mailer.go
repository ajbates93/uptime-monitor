package mailer

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

//go:embed "templates"
var templateFS embed.FS

type Mailer struct {
	apiKey string
	sender string
	client *http.Client
}

// SMTP2GO API request structure
type SMTP2GORequest struct {
	APIKey   string   `json:"api_key"`
	To       []string `json:"to"`
	Sender   string   `json:"sender"`
	Subject  string   `json:"subject"`
	TextBody string   `json:"text_body"`
	HtmlBody string   `json:"html_body"`
}

// SMTP2GO API response structure
type SMTP2GOResponse struct {
	RequestID string `json:"request_id"`
	Data      struct {
		EmailID string `json:"email_id"`
	} `json:"data"`
}

func New(apiKey, sender string) Mailer {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	return Mailer{
		apiKey: apiKey,
		sender: sender,
		client: client,
	}
}

func (m Mailer) Send(recipient, templateFile string, data any) error {
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	plainBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	htmlBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	// Prepare SMTP2GO API request
	request := SMTP2GORequest{
		APIKey:   m.apiKey,
		To:       []string{recipient},
		Sender:   m.sender,
		Subject:  subject.String(),
		TextBody: plainBody.String(),
		HtmlBody: htmlBody.String(),
	}

	// Convert request to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	fmt.Printf("SMTP2GO Request: %s\n", string(jsonData))

	// Send request to SMTP2GO API
	for i := 1; i <= 3; i++ {
		err = m.sendViaAPI(jsonData)
		if err == nil {
			return nil
		}

		fmt.Printf("SMTP2GO attempt %d failed: %v\n", i, err)

		// Wait before retry
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("failed to send email after 3 attempts: %w", err)
}

func (m Mailer) sendViaAPI(jsonData []byte) error {
	req, err := http.NewRequest("POST", "https://api.smtp2go.com/v3/email/send", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	// Parse response
	var response SMTP2GOResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}
