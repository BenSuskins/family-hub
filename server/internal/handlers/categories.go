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

type CategoryHandler struct {
	categoryRepo repository.CategoryRepository
}

func NewCategoryHandler(categoryRepo repository.CategoryRepository) *CategoryHandler {
	return &CategoryHandler{categoryRepo: categoryRepo}
}

func (handler *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	category := models.Category{
		Name:            r.FormValue("name"),
		CreatedByUserID: user.ID,
	}

	created, err := handler.categoryRepo.Create(ctx, category)
	if err != nil {
		slog.Error("creating category", "error", err)
		http.Error(w, "Error creating category", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		component := pages.CategoryRow(created)
		component.Render(ctx, w)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusFound)
}

func (handler *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	categoryID := chi.URLParam(r, "id")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	if err := handler.categoryRepo.Update(ctx, categoryID, r.FormValue("name")); err != nil {
		slog.Error("updating category", "error", err)
		http.Error(w, "Error updating category", http.StatusInternalServerError)
		return
	}

	category, err := handler.categoryRepo.FindByID(ctx, categoryID)
	if err != nil {
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}
	pages.CategoryRow(category).Render(ctx, w)
}

func (handler *CategoryHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	category, err := handler.categoryRepo.FindByID(ctx, id)
	if err != nil {
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}
	pages.CategoryEditForm(category).Render(ctx, w)
}

func (handler *CategoryHandler) CancelEdit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	category, err := handler.categoryRepo.FindByID(ctx, id)
	if err != nil {
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}
	pages.CategoryRow(category).Render(ctx, w)
}

func (handler *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	categoryID := chi.URLParam(r, "id")

	if err := handler.categoryRepo.Delete(ctx, categoryID); err != nil {
		slog.Error("deleting category", "error", err)
		http.Error(w, "Error deleting category", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusFound)
}
