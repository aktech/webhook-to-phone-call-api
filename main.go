package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	token := os.Getenv("TOKEN")
	if token == "" {
		slog.Error("TOKEN environment variable is required")
		os.Exit(1)
	}
	http.HandleFunc("/alert/", alertHandler(token))
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/twiml", twimlHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	slog.Info("starting server", "port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func alertHandler(expectedToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if strings.TrimPrefix(r.URL.Path, "/alert/") != expectedToken {
			slog.Warn("unauthorized alert attempt", "ip", r.RemoteAddr)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if err := triggerTwilioCall(r); err != nil {
			slog.Error("failed to trigger call", "error", err)
			http.Error(w, "Failed to trigger call", http.StatusInternalServerError)
			return
		}
		slog.Info("alert triggered", "ip", r.RemoteAddr)
		fmt.Fprintln(w, "Alert triggered successfully")
	}
}

func triggerTwilioCall(r *http.Request) error {
	accountSid, apiKeySid, apiKeySecret := os.Getenv("TWILIO_ACCOUNT_SID"), os.Getenv("TWILIO_API_KEY_SID"), os.Getenv("TWILIO_API_KEY_SECRET")
	fromNumber, toNumber := os.Getenv("TWILIO_FROM_NUMBER"), os.Getenv("ALERT_TO_NUMBER")
	if accountSid == "" || apiKeySid == "" || apiKeySecret == "" || fromNumber == "" || toNumber == "" {
		return fmt.Errorf("missing Twilio environment variables")
	}
	data := url.Values{}
	data.Set("To", toNumber)
	data.Set("From", fromNumber)
	data.Set("Twiml", `<Response><Say voice="alice">Alert triggered</Say></Response>`)
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Calls.json", accountSid)
	slog.Info("calling twilio API", "url", apiURL, "params", data.Encode())
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(apiKeySid + ":" + apiKeySecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("twilio API error: %d - %s", resp.StatusCode, string(body))
	}
	slog.Info("twilio call queued", "status", resp.StatusCode, "response", string(body))
	return nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

func twimlHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><Response><Say voice="alice">Alert triggered</Say></Response>`)
}
