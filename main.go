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
	"sync"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	token := os.Getenv("TOKEN")
	if token == "" {
		slog.Error("TOKEN environment variable is required")
		os.Exit(1)
	}

	// Log configured phone numbers (masked for privacy)
	fromNumber := os.Getenv("TWILIO_FROM_NUMBER")
	toNumbers := os.Getenv("ALERT_TO_NUMBER")
	if fromNumber != "" {
		maskedFrom := fromNumber
		if len(fromNumber) >= 7 {
			maskedFrom = fromNumber[:7] + "****"
		}
		slog.Info("configured caller id", "from", maskedFrom)
	}
	if toNumbers != "" {
		numbers := strings.Split(toNumbers, ",")
		var maskedNumbers []string
		for _, num := range numbers {
			num = strings.TrimSpace(num)
			if len(num) >= 7 {
				maskedNumbers = append(maskedNumbers, num[:7]+"****")
			} else {
				maskedNumbers = append(maskedNumbers, num)
			}
		}
		slog.Info("configured alert recipients", "count", len(numbers), "numbers", strings.Join(maskedNumbers, ", "))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/alert/", alertHandler(token))
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/twiml", twimlHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	slog.Info("starting server", "port", port)
	if err := http.ListenAndServe(":"+port, loggingMiddleware(mux)); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = r.Header.Get("Fly-Request-Id")
		}
		next.ServeHTTP(rw, r)
		duration := time.Since(start)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration_ms", duration.Milliseconds(),
			"size_bytes", rw.size,
			"remote_ip", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"referer", r.Referer(),
			"request_id", requestID,
			"proto", r.Proto,
			"host", r.Host,
		)
	})
}

func alertHandler(expectedToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		token := strings.TrimPrefix(r.URL.Path, "/alert/")
		if token != expectedToken {
			slog.Warn("unauthorized alert attempt", "provided_token_length", len(token))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if err := triggerTwilioCall(r); err != nil {
			slog.Error("twilio call failed", "error", err)
			http.Error(w, "Failed to trigger call", http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, "Alert triggered successfully")
	}
}

func triggerTwilioCall(r *http.Request) error {
	accountSid, apiKeySid, apiKeySecret := os.Getenv("TWILIO_ACCOUNT_SID"), os.Getenv("TWILIO_API_KEY_SID"), os.Getenv("TWILIO_API_KEY_SECRET")
	fromNumber, toNumbers := os.Getenv("TWILIO_FROM_NUMBER"), os.Getenv("ALERT_TO_NUMBER")
	if accountSid == "" || apiKeySid == "" || apiKeySecret == "" || fromNumber == "" || toNumbers == "" {
		return fmt.Errorf("missing Twilio environment variables")
	}
	numbers := strings.Split(toNumbers, ",")
	slog.Info("initiating calls", "count", len(numbers), "from", fromNumber)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []string
	successCount := 0

	for _, toNumber := range numbers {
		toNumber = strings.TrimSpace(toNumber)
		wg.Add(1)
		go func(num string) {
			defer wg.Done()
			if err := makeCall(accountSid, apiKeySid, apiKeySecret, fromNumber, num); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Sprintf("%s: %v", num, err))
				mu.Unlock()
				slog.Error("call failed", "to", num, "error", err)
			} else {
				mu.Lock()
				successCount++
				mu.Unlock()
				slog.Info("call queued", "to", num, "from", fromNumber)
			}
		}(toNumber)
	}

	wg.Wait()

	slog.Info("call batch complete", "success", successCount, "failed", len(errs), "total", len(numbers))
	if len(errs) > 0 {
		return fmt.Errorf("failed calls: %s", strings.Join(errs, "; "))
	}
	return nil
}

func makeCall(accountSid, apiKeySid, apiKeySecret, fromNumber, toNumber string) error {
	data := url.Values{}
	data.Set("To", toNumber)
	data.Set("From", fromNumber)
	data.Set("Twiml", `<Response><Say voice="alice">Alert triggered</Say></Response>`)
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Calls.json", accountSid)
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
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twilio API error: %d - %s", resp.StatusCode, string(body))
	}
	return nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

func twimlHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><Response><Say voice="alice">Alert triggered</Say></Response>`)
}
