package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/templates/pages"
	"github.com/go-chi/chi/v5"
)

type ChoreHandler struct {
	choreRepo    repository.ChoreRepository
	categoryRepo repository.CategoryRepository
	userRepo     repository.UserRepository
	choreService *services.ChoreService
}

func NewChoreHandler(
	choreRepo repository.ChoreRepository,
	categoryRepo repository.CategoryRepository,
	userRepo repository.UserRepository,
	choreService *services.ChoreService,
) *ChoreHandler {
	return &ChoreHandler{
		choreRepo:    choreRepo,
		categoryRepo: categoryRepo,
		userRepo:     userRepo,
		choreService: choreService,
	}
}

func (handler *ChoreHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	filter := repository.ChoreFilter{}

	if status := r.URL.Query().Get("status"); status != "" {
		s := models.ChoreStatus(status)
		filter.Status = &s
	}
	if assignedTo := r.URL.Query().Get("assigned_to"); assignedTo != "" {
		filter.AssignedToUser = &assignedTo
	}
	if categoryID := r.URL.Query().Get("category"); categoryID != "" {
		filter.CategoryID = &categoryID
	}

	chores, err := handler.choreRepo.FindAll(ctx, filter)
	if err != nil {
		slog.Error("finding chores", "error", err)
		http.Error(w, "Error loading chores", http.StatusInternalServerError)
		return
	}

	categories, err := handler.categoryRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding categories", "error", err)
	}

	users, err := handler.userRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding users", "error", err)
	}

	component := pages.ChoreList(pages.ChoreListProps{
		User:       user,
		Chores:     chores,
		Categories: categories,
		Users:      users,
		Filter:     filter,
	})
	component.Render(ctx, w)
}

func (handler *ChoreHandler) CreateForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	categories, err := handler.categoryRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding categories", "error", err)
	}

	component := pages.ChoreForm(pages.ChoreFormProps{
		User:       user,
		Categories: categories,
		IsEdit:     false,
	})
	component.Render(ctx, w)
}

func (handler *ChoreHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	chore := models.Chore{
		Name:            r.FormValue("name"),
		Description:     r.FormValue("description"),
		CreatedByUserID: user.ID,
		RecurrenceType:  models.RecurrenceType(r.FormValue("recurrence_type")),
		RecurrenceValue: r.FormValue("recurrence_value"),
		RecurOnComplete: r.FormValue("recur_on_complete") == "on",
	}

	if categoryID := r.FormValue("category_id"); categoryID != "" {
		chore.CategoryID = &categoryID
	}

	if dueDateStr := r.FormValue("due_date"); dueDateStr != "" {
		dueDate, err := time.Parse("2006-01-02", dueDateStr)
		if err == nil {
			chore.DueDate = &dueDate
		}
	}

	if dueTime := r.FormValue("due_time"); dueTime != "" {
		chore.DueTime = &dueTime
	}

	created, err := handler.choreRepo.Create(ctx, chore)
	if err != nil {
		slog.Error("creating chore", "error", err)
		http.Error(w, "Error creating chore", http.StatusInternalServerError)
		return
	}

	if _, err := handler.choreService.AssignNextUser(ctx, created); err != nil {
		slog.Error("assigning chore", "error", err)
	}

	http.Redirect(w, r, "/chores", http.StatusFound)
}

func (handler *ChoreHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)
	choreID := chi.URLParam(r, "id")

	chore, err := handler.choreRepo.FindByID(ctx, choreID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	categories, err := handler.categoryRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding categories", "error", err)
	}

	component := pages.ChoreForm(pages.ChoreFormProps{
		User:       user,
		Categories: categories,
		Chore:      &chore,
		IsEdit:     true,
	})
	component.Render(ctx, w)
}

func (handler *ChoreHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	choreID := chi.URLParam(r, "id")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	chore, err := handler.choreRepo.FindByID(ctx, choreID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	chore.Name = r.FormValue("name")
	chore.Description = r.FormValue("description")
	chore.RecurrenceType = models.RecurrenceType(r.FormValue("recurrence_type"))
	chore.RecurrenceValue = r.FormValue("recurrence_value")
	chore.RecurOnComplete = r.FormValue("recur_on_complete") == "on"

	if categoryID := r.FormValue("category_id"); categoryID != "" {
		chore.CategoryID = &categoryID
	} else {
		chore.CategoryID = nil
	}

	if dueDateStr := r.FormValue("due_date"); dueDateStr != "" {
		dueDate, err := time.Parse("2006-01-02", dueDateStr)
		if err == nil {
			chore.DueDate = &dueDate
		}
	} else {
		chore.DueDate = nil
	}

	if dueTime := r.FormValue("due_time"); dueTime != "" {
		chore.DueTime = &dueTime
	} else {
		chore.DueTime = nil
	}

	if err := handler.choreRepo.Update(ctx, chore); err != nil {
		slog.Error("updating chore", "error", err)
		http.Error(w, "Error updating chore", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/chores", http.StatusFound)
}

func (handler *ChoreHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	choreID := chi.URLParam(r, "id")

	if err := handler.choreRepo.Delete(ctx, choreID); err != nil {
		slog.Error("deleting chore", "error", err)
		http.Error(w, "Error deleting chore", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/chores", http.StatusFound)
}

func (handler *ChoreHandler) Complete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)
	choreID := chi.URLParam(r, "id")

	if err := handler.choreService.CompleteChore(ctx, choreID, user.ID); err != nil {
		slog.Error("completing chore", "error", err, "chore_id", choreID)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		chore, _ := handler.choreRepo.FindByID(ctx, choreID)
		component := pages.ChoreRow(chore, user)
		component.Render(ctx, w)
		return
	}

	http.Redirect(w, r, "/chores", http.StatusFound)
}
