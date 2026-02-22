package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/go-chi/chi/v5"
)

type APIHandler struct {
	choreRepo      repository.ChoreRepository
	userRepo       repository.UserRepository
	categoryRepo   repository.CategoryRepository
	assignmentRepo repository.ChoreAssignmentRepository
	tokenRepo      repository.APITokenRepository
}

func NewAPIHandler(
	choreRepo repository.ChoreRepository,
	userRepo repository.UserRepository,
	categoryRepo repository.CategoryRepository,
	assignmentRepo repository.ChoreAssignmentRepository,
	tokenRepo repository.APITokenRepository,
) *APIHandler {
	return &APIHandler{
		choreRepo:      choreRepo,
		userRepo:       userRepo,
		categoryRepo:   categoryRepo,
		assignmentRepo: assignmentRepo,
		tokenRepo:      tokenRepo,
	}
}

func (handler *APIHandler) ListChores(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	filter := repository.ChoreFilter{}

	if status := r.URL.Query().Get("status"); status != "" {
		s := models.ChoreStatus(status)
		filter.Status = &s
	}
	if assignedTo := r.URL.Query().Get("assigned_to"); assignedTo != "" {
		filter.AssignedToUser = &assignedTo
	}

	chores, err := handler.choreRepo.FindAll(ctx, filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load chores"})
		return
	}
	writeJSON(w, http.StatusOK, chores)
}

func (handler *APIHandler) GetChore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chore, err := handler.choreRepo.FindByID(ctx, chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "chore not found"})
		return
	}
	writeJSON(w, http.StatusOK, chore)
}

func (handler *APIHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	users, err := handler.userRepo.FindAll(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load users"})
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (handler *APIHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, err := handler.userRepo.FindByID(ctx, chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (handler *APIHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	categories, err := handler.categoryRepo.FindAll(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load categories"})
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

func (handler *APIHandler) DashboardStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	choresDueToday, _ := handler.choreRepo.FindDueToday(ctx)
	overdueChores, _ := handler.choreRepo.FindOverdueChores(ctx)

	stats := map[string]interface{}{
		"chores_due_today": len(choresDueToday),
		"chores_overdue":   len(overdueChores),
	}
	writeJSON(w, http.StatusOK, stats)
}

func (handler *APIHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	name := r.FormValue("name")
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	rawToken := generateToken()
	token := models.APIToken{
		Name:            name,
		TokenHash:       repository.HashToken(rawToken),
		CreatedByUserID: user.ID,
	}

	created, err := handler.tokenRepo.Create(ctx, token)
	if err != nil {
		slog.Error("creating token", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create token"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":    created.ID,
		"name":  created.Name,
		"token": rawToken,
	})
}

func (handler *APIHandler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	if err := handler.tokenRepo.Delete(ctx, id); err != nil {
		slog.Error("deleting token", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete token"})
		return
	}

	w.WriteHeader(http.StatusOK)
}

func generateToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
