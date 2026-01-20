package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrOTPExpired     = errors.New("OTP expired or not found")
	ErrOTPInvalid     = errors.New("invalid OTP")
	ErrSessionInvalid = errors.New("invalid or expired session")
)

type AuthService struct {
	pool            *pgxpool.Pool
	larkBearerToken string
	otpExpiry       time.Duration
	sessionExpiry   time.Duration
}

func NewAuthService(pool *pgxpool.Pool, larkBearerToken string) *AuthService {
	return &AuthService{
		pool:            pool,
		larkBearerToken: larkBearerToken,
		otpExpiry:       10 * time.Minute,
		sessionExpiry:   24 * time.Hour * 7, // 7 days
	}
}

// SendOTP generates and sends a 6-digit OTP to the user's email
func (s *AuthService) SendOTP(ctx context.Context, email string) error {
	// Generate 6-digit OTP
	otp, err := generateOTP(6)
	if err != nil {
		return fmt.Errorf("failed to generate OTP: %w", err)
	}

	expiresAt := time.Now().Add(s.otpExpiry)

	// Store OTP in database
	_, err = s.pool.Exec(ctx,
		`INSERT INTO otp_records (email, otp, expires_at)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (email) DO UPDATE SET otp = $2, expires_at = $3`,
		email, otp, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to store OTP: %w", err)
	}

	// Send OTP via Lark API
	if s.larkBearerToken != "" {
		if err := s.sendOTPViaLark(ctx, email, otp); err != nil {
			log.Printf("Failed to send OTP via Lark: %v", err)
			// Continue anyway - log the OTP for development
		}
	}

	// Log OTP for development (remove in production)
	log.Println("==========================================")
	log.Printf("🔐 OTP for %s: %s", email, otp)
	log.Printf("   Expires: %s", expiresAt.Format(time.RFC3339))
	log.Println("==========================================")

	return nil
}

// VerifyOTP verifies the OTP and returns true if valid
func (s *AuthService) VerifyOTP(ctx context.Context, email, otp string) (bool, error) {
	var storedOTP string
	var expiresAt time.Time

	err := s.pool.QueryRow(ctx,
		`SELECT otp, expires_at FROM otp_records WHERE email = $1`, email).
		Scan(&storedOTP, &expiresAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrOTPExpired
	}
	if err != nil {
		return false, fmt.Errorf("failed to get OTP: %w", err)
	}

	// Check expiry
	if time.Now().After(expiresAt) {
		// Clean up expired OTP
		s.pool.Exec(ctx, `DELETE FROM otp_records WHERE email = $1`, email)
		return false, ErrOTPExpired
	}

	// Check OTP
	if storedOTP != otp {
		return false, ErrOTPInvalid
	}

	// Delete used OTP
	s.pool.Exec(ctx, `DELETE FROM otp_records WHERE email = $1`, email)

	return true, nil
}

// CreateSession creates a new session for the user and returns the token
func (s *AuthService) CreateSession(ctx context.Context, email string) (string, error) {
	token := uuid.New().String()
	expiresAt := time.Now().Add(s.sessionExpiry)

	_, err := s.pool.Exec(ctx,
		`INSERT INTO sessions (token, email, expires_at) VALUES ($1, $2, $3)`,
		token, email, expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return token, nil
}

// ValidateSession validates a session token and returns the user email
func (s *AuthService) ValidateSession(ctx context.Context, token string) (string, error) {
	var email string
	var expiresAt time.Time

	err := s.pool.QueryRow(ctx,
		`SELECT email, expires_at FROM sessions WHERE token = $1`, token).
		Scan(&email, &expiresAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrSessionInvalid
	}
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	// Check expiry
	if time.Now().After(expiresAt) {
		s.pool.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
		return "", ErrSessionInvalid
	}

	return email, nil
}

// DeleteSession deletes a session (logout)
func (s *AuthService) DeleteSession(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// CleanExpiredSessions removes expired sessions from the database
func (s *AuthService) CleanExpiredSessions(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < NOW()`)
	if err != nil {
		return fmt.Errorf("failed to clean sessions: %w", err)
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM otp_records WHERE expires_at < NOW()`)
	if err != nil {
		return fmt.Errorf("failed to clean OTP records: %w", err)
	}
	return nil
}

// sendOTPViaLark sends the OTP via Lark API
func (s *AuthService) sendOTPViaLark(ctx context.Context, email, otp string) error {
	// Prepare the request body
	requestBody := map[string]interface{}{
		"receive_id": email,
		"msg_type":   "text",
		"content":    fmt.Sprintf("{\"text\":\" THE OTP CODE: %s\"}", otp),
		"uuid":       uuid.New().String(),
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://open.larksuite.com/open-apis/im/v1/messages?receive_id_type=email",
		bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.larkBearerToken))

	// Send request with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		return fmt.Errorf("lark API returned status %d: %v", resp.StatusCode, result)
	}

	return nil
}

// generateOTP generates a random numeric OTP of the specified length
func generateOTP(length int) (string, error) {
	const digits = "0123456789"
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		result[i] = digits[num.Int64()]
	}

	return string(result), nil
}
