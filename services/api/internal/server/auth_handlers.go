package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"clientpulse/services/api/internal/service"
	"clientpulse/services/api/internal/tenant"
)

type AuthHandler struct {
	authService *service.AuthService
	isProd      bool
}

func NewAuthHandler(authService *service.AuthService, isProd bool) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		isProd:      isProd,
	}
}

// setAuthCookies sets the session HttpOnly cookie and CSRF token cookie.
func (h *AuthHandler) setAuthCookies(w http.ResponseWriter, sessionToken, csrfToken string) {
	// Session Cookie: HttpOnly, Secure (in prod), SameSite=Lax
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		MaxAge:   7 * 24 * 3600,
		HttpOnly: true,
		Secure:   h.isProd,
		SameSite: http.SameSiteLaxMode,
	})

	// CSRF Cookie: accessible by JavaScript to place into X-CSRF-Token header
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    csrfToken,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		MaxAge:   7 * 24 * 3600,
		HttpOnly: false,
		Secure:   h.isProd,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearAuthCookies removes session and CSRF cookies.
func (h *AuthHandler) clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.isProd,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   h.isProd,
		SameSite: http.SameSiteLaxMode,
	})
}

// Signup handles new user registration.
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req service.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload format.")
		return
	}

	req.IPAddress = getClientIP(r)
	req.UserAgent = r.UserAgent()

	result, err := h.authService.Signup(r.Context(), req)
	if err != nil {
		sendJSONError(w, http.StatusBadRequest, "SIGNUP_FAILED", err.Error())
		return
	}

	h.setAuthCookies(w, result.RawSession, result.RawCSRF)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(result)
}

// Login authenticates a user with email and password.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload format.")
		return
	}

	ip := getClientIP(r)
	userAgent := r.UserAgent()

	result, err := h.authService.Login(r.Context(), req.Email, req.Password, ip, userAgent)
	if err != nil {
		if errors.Is(err, service.ErrUserSuspended) {
			sendJSONError(w, http.StatusForbidden, "USER_SUSPENDED", "Your account has been suspended by platform administration.")
			return
		}
		sendJSONError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password.")
		return
	}

	h.setAuthCookies(w, result.RawSession, result.RawCSRF)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

// Logout revokes the current session and clears cookies.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	u := tenant.GetUserContext(r.Context())
	if u != nil && u.SessionID != "" {
		_ = h.authService.Logout(r.Context(), u.SessionID)
	}

	h.clearAuthCookies(w)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Logged out successfully.",
	})
}

// Me returns the authenticated user's profile and active organization.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	u := tenant.GetUserContext(r.Context())
	if u == nil {
		sendJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required.")
		return
	}

	result, err := h.authService.GetMe(r.Context(), u.UserID, u.SessionID)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve profile.")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

// VerifyEmail verifies user email with token.
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.Token == "" {
		req.Token = r.URL.Query().Get("token")
	}

	if req.Token == "" {
		sendJSONError(w, http.StatusBadRequest, "INVALID_TOKEN", "Verification token is required.")
		return
	}

	if err := h.authService.VerifyEmail(r.Context(), req.Token); err != nil {
		sendJSONError(w, http.StatusBadRequest, "VERIFICATION_FAILED", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Email verified successfully.",
	})
}

// ResendVerification sends a fresh verification email.
func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	u := tenant.GetUserContext(r.Context())
	if u == nil {
		sendJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required.")
		return
	}

	if err := h.authService.SendVerificationEmail(r.Context(), u.UserID, u.Email, u.DisplayName); err != nil {
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to send verification email.")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Verification email sent.",
	})
}

// ForgotPassword initiates password recovery with anti-enumeration protection.
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload format.")
		return
	}

	_ = h.authService.ForgotPassword(r.Context(), req.Email)

	// Always return identical success response to prevent email enumeration
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "If your email is registered, you will receive password reset instructions shortly.",
	})
}

// ResetPassword resets user password using valid token.
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload format.")
		return
	}

	if err := h.authService.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
		sendJSONError(w, http.StatusBadRequest, "RESET_FAILED", err.Error())
		return
	}

	// Password reset invalidates sessions; clear cookies
	h.clearAuthCookies(w)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Your password has been reset successfully. Please log in with your new password.",
	})
}

// ChangePassword changes password for authenticated user.
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	u := tenant.GetUserContext(r.Context())
	if u == nil {
		sendJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required.")
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload format.")
		return
	}

	if err := h.authService.ChangePassword(r.Context(), u.UserID, req.CurrentPassword, req.NewPassword); err != nil {
		sendJSONError(w, http.StatusBadRequest, "CHANGE_PASSWORD_FAILED", err.Error())
		return
	}

	// Password change invalidates previous sessions; user must re-authenticate
	h.clearAuthCookies(w)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Password changed successfully. Please log in again.",
	})
}

func sendJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func sendJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	})
}

