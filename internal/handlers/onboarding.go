package handlers

import (
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/templates/components"
	"github.com/bensuskins/family-hub/templates/pages"
)

type OnboardingHandler struct {
	settingsRepo repository.SettingsRepository
	userRepo     repository.UserRepository
	categoryRepo repository.CategoryRepository
}

func NewOnboardingHandler(
	settingsRepo repository.SettingsRepository,
	userRepo repository.UserRepository,
	categoryRepo repository.CategoryRepository,
) *OnboardingHandler {
	return &OnboardingHandler{
		settingsRepo: settingsRepo,
		userRepo:     userRepo,
		categoryRepo: categoryRepo,
	}
}

func (handler *OnboardingHandler) SetupPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	familyName, _ := handler.settingsRepo.Get(ctx, "family_name")
	if familyName == "" {
		familyName = "Family"
	}

	component := pages.Setup(pages.SetupProps{User: user, FamilyName: familyName})
	component.Render(ctx, w)
}

func (handler *OnboardingHandler) SaveFamilyName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("family_name"))
	if name == "" {
		name = "Family"
	}

	if err := handler.settingsRepo.Set(ctx, "family_name", name); err != nil {
		slog.Error("saving family name", "error", err)
		http.Error(w, "Error saving family name", http.StatusInternalServerError)
		return
	}

	allUsers, err := handler.userRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding users", "error", err)
		allUsers = []models.User{user}
	}

	component := components.SetupStepUsers(components.SetupStepUsersProps{Users: allUsers})
	component.Render(ctx, w)
}

func (handler *OnboardingHandler) AcknowledgeUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	component := components.SetupStepCategory()
	component.Render(ctx, w)
}

func (handler *OnboardingHandler) CompleteSetup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	categoryName := strings.TrimSpace(r.FormValue("category_name"))
	if categoryName != "" {
		if _, err := handler.categoryRepo.Create(ctx, models.Category{
			Name:            categoryName,
			CreatedByUserID: user.ID,
		}); err != nil {
			slog.Error("creating first category", "error", err)
		}
	}

	if err := handler.settingsRepo.Set(ctx, "onboarding_complete", "true"); err != nil {
		slog.Error("setting onboarding_complete", "error", err)
		http.Error(w, "Error completing setup", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func (handler *OnboardingHandler) WelcomePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	familyName, _ := handler.settingsRepo.Get(ctx, "family_name")
	if familyName == "" {
		familyName = "Family"
	}

	component := pages.Welcome(pages.WelcomeProps{User: user, FamilyName: familyName})
	component.Render(ctx, w)
}

func (handler *OnboardingHandler) WelcomeStart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)
	component := components.WelcomeStepProfile(components.WelcomeStepProfileProps{User: user})
	component.Render(ctx, w)
}

func (handler *OnboardingHandler) CompleteWelcome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	if err := r.ParseMultipartForm(1*1024*1024 + 1024); err != nil {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = user.Name
	}

	if err := handler.userRepo.UpdateProfile(ctx, user.ID, name, user.Email, user.AvatarURL); err != nil {
		slog.Error("updating profile during welcome", "error", err)
		http.Error(w, "Error saving profile", http.StatusInternalServerError)
		return
	}

	if file, header, err := r.FormFile("avatar"); err == nil {
		defer file.Close()
		const maxBytes = 1 * 1024 * 1024
		imageBytes, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
		if err == nil && len(imageBytes) <= maxBytes {
			contentType := header.Header.Get("Content-Type")
			if contentType == "" {
				contentType = http.DetectContentType(imageBytes)
			}
			encoded := base64.StdEncoding.EncodeToString(imageBytes)
			dataURI := "data:" + contentType + ";base64," + encoded
			if avatarErr := handler.userRepo.UpdateAvatar(ctx, user.ID, dataURI); avatarErr != nil {
				slog.Error("updating avatar during welcome", "error", avatarErr)
			}
		}
	}

	if err := handler.userRepo.MarkOnboarded(ctx, user.ID); err != nil {
		slog.Error("marking user as onboarded", "error", err)
		http.Error(w, "Error completing welcome", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
