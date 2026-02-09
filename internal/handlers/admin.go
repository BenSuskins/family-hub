package handlers

import (
	"log/slog"
	"net/http"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/templates/pages"
	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	userRepo  repository.UserRepository
	tokenRepo repository.APITokenRepository
}

func NewAdminHandler(userRepo repository.UserRepository, tokenRepo repository.APITokenRepository) *AdminHandler {
	return &AdminHandler{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
	}
}

func (handler *AdminHandler) Users(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	users, err := handler.userRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding users", "error", err)
		http.Error(w, "Error loading users", http.StatusInternalServerError)
		return
	}

	tokens, err := handler.tokenRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding tokens", "error", err)
	}

	component := pages.AdminUsers(pages.AdminUsersProps{
		User:      user,
		AllUsers:  users,
		APITokens: tokens,
	})
	component.Render(ctx, w)
}

func (handler *AdminHandler) PromoteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "id")

	if err := handler.userRepo.UpdateRole(ctx, userID, models.RoleAdmin); err != nil {
		slog.Error("promoting user", "error", err)
		http.Error(w, "Error promoting user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusFound)
}

func (handler *AdminHandler) DemoteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "id")

	if err := handler.userRepo.UpdateRole(ctx, userID, models.RoleMember); err != nil {
		slog.Error("demoting user", "error", err)
		http.Error(w, "Error demoting user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusFound)
}
