package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/internal/auth"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

type PutUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type PostLoginRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type PostLoginResponse struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	AccessToken string    `json:"token"`
	SessionID   string    `json:"refresh_token"`
}

type SessionRefreshResponse struct {
	AccessToken string `json:"token"`
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var params PostLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Cannot decode User from request body", http.StatusBadRequest)
		return
	}
	email := params.Email
	hashedPassword, err := auth.HashPassword(params.Password)
	user, err := s.Repositories.Users.CreateUser(r.Context(), email, hashedPassword)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			respondWithMessage(w, http.StatusConflict, "User already exists")
		} else {
			respondWithMessage(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	createUserResponse := PostLoginResponse{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}

	respondWithJSON(w, http.StatusCreated, createUserResponse)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	/* Pseudocode (reverse engineering from the course author's implicit requirement)
		if we have an access token, then
		  if token is valid (returns UUID of the user)
	    	look for the user record by its UUID
			if the record is found, then
			    if we can create the user with this new email (collision risk here!), then
					delete the old user record
					return new user record
				otherwise
					return the bad news about email collision
			otherwise (... odd, because we know they existed when they got a good token)
			   return not found
		  otherwise
		    return unauthorized
		otherwise
		  return unauthorized
	*/

	accessToken, err := auth.GetAccessToken(r.Header)
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Error retrieving access token: %s", err.Error()),
			http.StatusUnauthorized,
		)
		return
	}

	user_id, err := auth.ValidateJWT(accessToken, s.environment.SecretKey)
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Error validating access token: %s", err.Error()),
			http.StatusUnauthorized,
		)
		return
	}

	var params PutUserRequest
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(
			w,
			fmt.Sprintf("Error decoding PutUserRequest: %s", err.Error()),
			http.StatusUnauthorized,
		)
		return
	}

	oldUser, err := s.Repositories.Users.GetUserByID(r.Context(), user_id.String())
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Error locating you by your UUID: %s", err.Error()),
			http.StatusUnauthorized,
		)
		return
	}

	email := params.Email
	hashedPassword, err := auth.HashPassword(params.Password)
	newUser, err := s.Repositories.Users.CreateUser(r.Context(), email, hashedPassword)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			respondWithMessage(w, http.StatusConflict, "User already exists")
		} else {
			respondWithMessage(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	// we're trusting they know what they're doing here (sigh)
	s.Repositories.Users.DeleteUser(r.Context(), oldUser.Email)

	createUserResponse := PostLoginResponse{
		ID:        newUser.ID,
		CreatedAt: newUser.CreatedAt,
		UpdatedAt: newUser.UpdatedAt,
		Email:     newUser.Email,
	}

	respondWithJSON(w, http.StatusOK, createUserResponse)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var loginRequestBody PostLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginRequestBody); err != nil {
		http.Error(w, "Cannot decode User from request body", http.StatusBadRequest)
		return
	}

	user, err := s.Repositories.Users.AuthenticateUser(
		r.Context(), loginRequestBody.Email, loginRequestBody.Password)
	if err != nil {
		fmt.Printf("User authentication error: %v\n", err)
		if errors.Is(err, domain.ErrNotFound) {
			respondWithMessage(w, http.StatusNotFound, "User not found")
		} else {
			respondWithMessage(w, http.StatusUnauthorized, "Not authorized")
		}
		return
	}

	// choose the minimum of 3600 seconds and the user's requested expires_in_seconds
	minExpiry := 3600
	if loginRequestBody.ExpiresInSeconds > 0 && loginRequestBody.ExpiresInSeconds < minExpiry {
		minExpiry = loginRequestBody.ExpiresInSeconds
	}

	token, err := auth.MakeJWT(
		user.ID,
		s.environment.SecretKey,
		time.Duration(minExpiry)*time.Second,
	)
	if err != nil {
		respondWithMessage(w, http.StatusInternalServerError, "Server error creating JWT")
		return
	}

	session, err := s.Repositories.UserSessions.CreateSession(r.Context(), user.ID)
	if err != nil {
		respondWithMessage(w, http.StatusInternalServerError, "Server error creating User Session")
		return
	}

	loginReply := PostLoginResponse{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		AccessToken: token,
		SessionID:   session.ID,
	}

	respondWithJSON(w, http.StatusOK, loginReply)
}

func (s *Server) handleSessionRefresh(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetRefreshToken(r.Header)
	if err != nil {
		respondWithMessage(
			w,
			http.StatusBadRequest,
			fmt.Errorf("Cannot retrieve refresh token: %s", err).Error(),
		)
		return
	}

	session, err := s.Repositories.UserSessions.GetSession(r.Context(), refreshToken)
	if err != nil {
		respondWithMessage(
			w,
			http.StatusUnauthorized,
			fmt.Errorf("Cannot retrieve session: %s", err).Error(),
		)
		return
	}

	if session.ExpiresAt.Before(time.Now()) {
		respondWithMessage(
			w,
			http.StatusUnauthorized,
			"Session token has expired. Please re-authenticate.",
		)
		return
	}

	if !session.RevokedAt.IsZero() {
		respondWithMessage(
			w,
			http.StatusUnauthorized,
			"Session token was revoked. Please re-authenticate.",
		)
		return
	}

	accessToken, err := auth.MakeJWT(
		session.UserID,
		s.environment.SecretKey,
		time.Duration(3600)*time.Second,
	)
	if err != nil {
		respondWithMessage(
			w,
			http.StatusInternalServerError,
			fmt.Errorf("Error creating access token: %s", err).Error(),
		)
		return
	}

	refreshTokenResponse := SessionRefreshResponse{AccessToken: accessToken}
	respondWithJSON(w, http.StatusOK, refreshTokenResponse)
}

func (s *Server) handleSessionRevoke(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetRefreshToken(r.Header)
	if err != nil {
		respondWithMessage(
			w,
			http.StatusBadRequest,
			fmt.Errorf("Cannot retrieve refresh token: %s", err).Error(),
		)
		return
	}

	err = s.Repositories.UserSessions.RevokeSession(r.Context(), refreshToken)
	if err != nil {
		respondWithMessage(
			w,
			http.StatusBadRequest,
			fmt.Errorf("Cannot revoke session: %s", err).Error(),
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
