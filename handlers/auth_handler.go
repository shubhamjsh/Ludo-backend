package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"Ludo/models"
	"Ludo/repository"
	"Ludo/utils"
)

type AuthHandler struct {
	userRepo *repository.UserRepository
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		userRepo: repository.NewUserRepository(),
	}
}

// Signup handles user registration
// @Summary Register a new user
// @Description Create a new user account with email/phone and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.SignupRequest true "Signup Request"
// @Success 201 {object} utils.Response{data=models.AuthResponse} "User created successfully"
// @Failure 400 {object} utils.Response "Invalid request"
// @Failure 409 {object} utils.Response "Email or phone already exists"
// @Failure 500 {object} utils.Response "Internal server error"
// @Router /api/auth/signup [post]
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req models.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate that at least email or phone is provided
	if req.Email == nil && req.Phone == nil {
		utils.SendError(w, http.StatusBadRequest, "Either email or phone is required")
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error processing password")
		return
	}

	// Generate default avatar URL
	avatarURL := fmt.Sprintf("https://ui-avatars.com/api/?name=%s&background=random", req.Name)

	// Create user
	user := &models.User{
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		Password:  &hashedPassword,
		AvatarURL: avatarURL,
		Coins:     1000, // Starting coins
		IsGuest:   false,
		CreatedAt: time.Now(),
	}

	if err := h.userRepo.CreateUser(user); err != nil {
		if err == repository.ErrDuplicateEmail {
			utils.SendError(w, http.StatusConflict, "Email already exists")
			return
		}
		if err == repository.ErrDuplicatePhone {
			utils.SendError(w, http.StatusConflict, "Phone already exists")
			return
		}
		utils.SendError(w, http.StatusInternalServerError, "Error creating user")
		return
	}

	// Update last login
	h.userRepo.UpdateLastLogin(user.ID)

	// Generate JWT token
	token, err := utils.GenerateToken(user.ID, false)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error generating token")
		return
	}

	// Clear password before sending response
	user.Password = nil

	utils.SendCreated(w, "User created successfully", models.AuthResponse{
		Token: token,
		User:  *user,
	})
}

// Login handles user authentication
// @Summary Login with credentials
// @Description Authenticate user with email/phone and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login Request"
// @Success 200 {object} utils.Response{data=models.AuthResponse} "Login successful"
// @Failure 400 {object} utils.Response "Invalid request"
// @Failure 401 {object} utils.Response "Invalid credentials"
// @Failure 500 {object} utils.Response "Internal server error"
// @Router /api/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorLogger.Printf("Failed to decode login request: %v", err)
		utils.SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	utils.InfoLogger.Printf("Login attempt for: %s", req.EmailOrPhone)
	// Try to find user by email or phone
	var user *models.User
	var err error

	if strings.Contains(req.EmailOrPhone, "@") {
		user, err = h.userRepo.GetUserByEmail(req.EmailOrPhone)
	} else {
		user, err = h.userRepo.GetUserByPhone(req.EmailOrPhone)
	}

	if err != nil {
		utils.SendError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Check if it's a guest account
	if user.IsGuest {
		utils.SendError(w, http.StatusBadRequest, "Cannot login to guest account with password")
		return
	}

	// Verify password
	if user.Password == nil || !utils.CheckPassword(*user.Password, req.Password) {
		utils.SendError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Update last login
	h.userRepo.UpdateLastLogin(user.ID)

	// Generate JWT token
	token, err := utils.GenerateToken(user.ID, false)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error generating token")
		return
	}

	// Clear password before sending response
	user.Password = nil

	utils.SendSuccess(w, "Login successful", models.AuthResponse{
		Token: token,
		User:  *user,
	})
}

// GuestLogin handles guest user creation and authentication
// @Summary Create guest account
// @Description Create a temporary guest account without email/password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.GuestLoginRequest true "Guest Login Request"
// @Success 201 {object} utils.Response{data=models.AuthResponse} "Guest login successful"
// @Failure 400 {object} utils.Response "Invalid request"
// @Failure 500 {object} utils.Response "Internal server error"
// @Router /api/auth/guest-login [post]
func (h *AuthHandler) GuestLogin(w http.ResponseWriter, r *http.Request) {
	var req models.GuestLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Generate random guest name
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	guestName := "Guest_" + hex.EncodeToString(randomBytes)

	// Generate avatar
	avatarURL := fmt.Sprintf("https://ui-avatars.com/api/?name=%s&background=random", guestName)

	// Create guest user
	user := &models.User{
		Name:      guestName,
		AvatarURL: avatarURL,
		Coins:     500, // Guests start with fewer coins
		IsGuest:   true,
		CreatedAt: time.Now(),
	}

	if err := h.userRepo.CreateUser(user); err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error creating guest user")
		return
	}

	// Update last login
	h.userRepo.UpdateLastLogin(user.ID)

	// Generate JWT token
	token, err := utils.GenerateToken(user.ID, true)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error generating token")
		return
	}

	utils.SendCreated(w, "Guest login successful", models.AuthResponse{
		Token: token,
		User:  *user,
	})
}
