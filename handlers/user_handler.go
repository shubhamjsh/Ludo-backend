package handlers

import (
	"encoding/json"
	"net/http"

	"Ludo/models"
	"Ludo/repository"
	"Ludo/utils"
)

type UserHandler struct {
	userRepo *repository.UserRepository
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		userRepo: repository.NewUserRepository(),
	}
}

// GetProfile retrieves the authenticated user's profile
// @Summary Get user profile
// @Description Retrieve the authenticated user's profile information
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.Response{data=models.User} "Profile retrieved successfully"
// @Failure 401 {object} utils.Response "Unauthorized"
// @Failure 404 {object} utils.Response "User not found"
// @Failure 500 {object} utils.Response "Internal server error"
// @Router /api/user/profile [get]
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		utils.SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.userRepo.GetUserByID(userID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			utils.SendError(w, http.StatusNotFound, "User not found")
			return
		}
		utils.SendError(w, http.StatusInternalServerError, "Error fetching profile")
		return
	}

	// Clear password before sending
	user.Password = nil

	utils.SendSuccess(w, "Profile retrieved successfully", user)
}

// UpdateProfile updates the authenticated user's profile
// @Summary Update user profile
// @Description Update the authenticated user's name and avatar
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UpdateProfileRequest true "Update Profile Request"
// @Success 200 {object} utils.Response{data=models.User} "Profile updated successfully"
// @Failure 400 {object} utils.Response "Invalid request"
// @Failure 401 {object} utils.Response "Unauthorized"
// @Failure 500 {object} utils.Response
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		utils.SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update profile
	err := h.userRepo.UpdateProfile(userID, req.Name, req.AvatarURL)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error updating profile")
		return
	}

	// Fetch updated user
	user, err := h.userRepo.GetUserByID(userID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error fetching updated profile")
		return
	}

	// Clear password before sending
	user.Password = nil

	utils.SendSuccess(w, "Profile updated successfully", user)
}
