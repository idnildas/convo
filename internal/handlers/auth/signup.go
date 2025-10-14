package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"convo/internal/utils"

)

type SignupHandler struct {
	DB *sql.DB
}

type SignupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignupResponse struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// ServeHTTP handles POST /signup
func (h *SignupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// decode request
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, utils.APIResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// hash password
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{
			Success: false,
			Message: "Failed to hash password",
		})
		return
	}

	// insert into DB
	result, err := h.DB.Exec(
		"INSERT INTO users (name, email, password_hash) VALUES (?, ?, ?)",
		req.Name, req.Email, hash,
	)
	if err != nil {
		utils.JSON(w, http.StatusBadRequest, utils.APIResponse{
			Success: false,
			Message: "Could not create user (maybe duplicate email)",
		})
		return
	}

	id, _ := result.LastInsertId()

	resp := SignupResponse{
		ID:    id,
		Email: req.Email,
		Name:  req.Name,
	}

	utils.JSON(w, http.StatusCreated, utils.APIResponse{
		Success: true,
		Message: "User created successfully",
		Data:    resp,
	})
}
