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
	userRepo       repository.UserRepository
	tokenRepo      repository.APITokenRepository
	settingsRepo   repository.SettingsRepository
	categoryRepo   repository.CategoryRepository
	assignmentRepo repository.ChoreAssignmentRepository
}

func NewAdminHandler(
	userRepo repository.UserRepository,
	tokenRepo repository.APITokenRepository,
	settingsRepo repository.SettingsRepository,
	categoryRepo repository.CategoryRepository,
	assignmentRepo repository.ChoreAssignmentRepository,
) *AdminHandler {
	return &AdminHandler{
		userRepo:       userRepo,
		tokenRepo:      tokenRepo,
		settingsRepo:   settingsRepo,
		categoryRepo:   categoryRepo,
		assignmentRepo: assignmentRepo,
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

	categories, err := handler.categoryRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding categories", "error", err)
	}

	familyName, err := handler.settingsRepo.Get(ctx, "family_name")
	if err != nil {
		slog.Error("getting family name", "error", err)
		familyName = "Family"
	}

	component := pages.AdminUsers(pages.AdminUsersProps{
		User:       user,
		AllUsers:   users,
		APITokens:  tokens,
		Categories: categories,
		FamilyName: familyName,
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

func (handler *AdminHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	scope := r.FormValue("scope")
	if scope != "api" && scope != "ical" {
		scope = "api"
	}

	rawToken := generateToken()
	token := models.APIToken{
		Name:            name,
		Scope:           scope,
		TokenHash:       repository.HashToken(rawToken),
		CreatedByUserID: user.ID,
	}

	if _, err := handler.tokenRepo.Create(ctx, token); err != nil {
		slog.Error("creating token", "error", err)
		http.Error(w, "failed to create token", http.StatusInternalServerError)
		return
	}

	pages.AdminTokenCreated(name, rawToken).Render(ctx, w)
}

func (handler *AdminHandler) DeleteChoreHistory(w http.ResponseWriter, r *http.Request) {
	if err := handler.assignmentRepo.DeleteCompleted(r.Context()); err != nil {
		slog.Error("deleting chore history", "error", err)
		http.Error(w, "failed to delete chore history", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusFound)
}

func (handler *AdminHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	familyName := r.FormValue("family_name")
	if familyName != "" {
		if err := handler.settingsRepo.Set(ctx, "family_name", familyName); err != nil {
			slog.Error("updating family name", "error", err)
			http.Error(w, "Error updating settings", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/admin/users", http.StatusFound)
}
