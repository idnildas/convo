package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"convo/internal/utils"
)

type LoginHandler struct {
	DB        *sql.DB
	JWTSecret string
	JWTTTLHrs int
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// ServeHTTP handles POST /auth/login
func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, utils.APIResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// 1. Find user
	var id int64
	var name, passwordHash string
	err := h.DB.QueryRow("SELECT id, name, password_hash FROM users WHERE email=?", req.Email).Scan(&id, &name, &passwordHash)
	if err == sql.ErrNoRows {
		utils.JSON(w, http.StatusUnauthorized, utils.APIResponse{
			Success: false,
			Message: "Invalid email or password",
		})
		return
	} else if err != nil {
		utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{
			Success: false,
			Message: "Database error",
		})
		return
	}

	// 2. Verify password
	if !utils.CheckPassword(req.Password, passwordHash) {
		utils.JSON(w, http.StatusUnauthorized, utils.APIResponse{
			Success: false,
			Message: "Invalid email or password",
		})
		return
	}

	// 3. Generate JWT
	token, err := utils.GenerateJWT(id, h.JWTSecret, h.JWTTTLHrs)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{
			Success: false,
			Message: "Failed to generate token",
		})
		return
	}

	//save the token in user_tokens table if needed
	// have this schema :
	// CREATE TABLE IF NOT EXISTS user_tokens (
	// 	id BIGINT AUTO_INCREMENT PRIMARY KEY,
	// 	user_id BIGINT NOT NULL,
	// 	token CHAR(64) NOT NULL UNIQUE,
	// 	token_type ENUM('verify','reset') NOT NULL,
	// 	expires_at DATETIME NOT NULL,
	// 	used_at DATETIME NULL,
	// 	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	// 	CONSTRAINT fk_tokens_user FOREIGN KEY (user_id) REFERENCES users(id)
	// 	  ON DELETE CASCADE ON UPDATE CASCADE
	//   ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

	_, err = h.DB.Exec("INSERT INTO user_tokens (user_id, token, token_type, expires_at) VALUES (?, ?, 'verify', ?)", id, token, time.Now().Add(time.Duration(h.JWTTTLHrs)*time.Hour))
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{
			Success: false,
			Message: "Failed to save token",
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		})
		return
	}


	// 4. Return response
	resp := LoginResponse{
		Token: token,
		Email: req.Email,
		Name:  name,
	}
	utils.JSON(w, http.StatusOK, utils.APIResponse{
		Success: true,
		Message: "Login successful",
		Data:    resp,
	})
}
