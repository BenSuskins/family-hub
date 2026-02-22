package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
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

	tab := r.URL.Query().Get("tab")
	if tab == "" {
		tab = "active"
	}

	filter := repository.ChoreFilter{}

	if assignedTo := r.URL.Query().Get("assigned_to"); assignedTo != "" {
		filter.AssignedToUser = &assignedTo
	}
	if categoryID := r.URL.Query().Get("category"); categoryID != "" {
		filter.CategoryID = &categoryID
	}

	users, err := handler.userRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding users", "error", err)
	}

	userNameMap := make(map[string]string, len(users))
	userAvatarMap := make(map[string]string, len(users))
	for _, u := range users {
		userNameMap[u.ID] = u.Name
		userAvatarMap[u.ID] = u.AvatarURL
	}

	if tab == "history" {
		filter.Statuses = []models.ChoreStatus{models.ChoreStatusCompleted}
		filter.OrderBy = repository.OrderByCompletedAtDesc

		chores, err := handler.choreRepo.FindAll(ctx, filter)
		if err != nil {
			slog.Error("finding chores", "error", err)
			http.Error(w, "Error loading chores", http.StatusInternalServerError)
			return
		}

		historyEntries := buildHistoryEntries(chores, userNameMap)

		if r.Header.Get("HX-Request") == "true" {
			component := pages.ChoreHistoryContent(historyEntries)
			component.Render(ctx, w)
			return
		}

		categories, err := handler.categoryRepo.FindAll(ctx)
		if err != nil {
			slog.Error("finding categories", "error", err)
		}

		component := pages.ChoreList(pages.ChoreListProps{
			User:           user,
			HistoryEntries: historyEntries,
			Categories:     categories,
			Users:          users,
			UserNameMap:    userNameMap,
			UserAvatarMap:  userAvatarMap,
			Filter:         filter,
			ActiveTab:      tab,
		})
		component.Render(ctx, w)
		return
	}

	if status := r.URL.Query().Get("status"); status != "" {
		s := models.ChoreStatus(status)
		filter.Status = &s
	} else {
		filter.Statuses = []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue}
	}

	chores, err := handler.choreRepo.FindAll(ctx, filter)
	if err != nil {
		slog.Error("finding chores", "error", err)
		http.Error(w, "Error loading chores", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		component := pages.ChoreTableContent(pages.ChoreTableProps{
			Chores:        chores,
			User:          user,
			UserNameMap:   userNameMap,
			UserAvatarMap: userAvatarMap,
		})
		component.Render(ctx, w)
		return
	}

	categories, err := handler.categoryRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding categories", "error", err)
	}

	component := pages.ChoreList(pages.ChoreListProps{
		User:          user,
		Chores:        chores,
		Categories:    categories,
		Users:         users,
		UserNameMap:   userNameMap,
		UserAvatarMap: userAvatarMap,
		Filter:        filter,
		ActiveTab:     tab,
	})
	component.Render(ctx, w)
}

func buildHistoryEntries(chores []models.Chore, userNameMap map[string]string) []pages.ChoreHistoryEntry {
	type accumulator struct {
		count           int
		lastCompletedAt *time.Time
		lastCompletedBy string
	}

	grouped := make(map[string]*accumulator)
	for _, chore := range chores {
		entry, exists := grouped[chore.Name]
		if !exists {
			entry = &accumulator{}
			grouped[chore.Name] = entry
		}
		entry.count++
		if chore.CompletedAt != nil && (entry.lastCompletedAt == nil || chore.CompletedAt.After(*entry.lastCompletedAt)) {
			entry.lastCompletedAt = chore.CompletedAt
			if chore.CompletedByUserID != nil {
				entry.lastCompletedBy = userNameMap[*chore.CompletedByUserID]
			}
		}
	}

	entries := make([]pages.ChoreHistoryEntry, 0, len(grouped))
	for name, acc := range grouped {
		entries = append(entries, pages.ChoreHistoryEntry{
			Name:            name,
			CompletionCount: acc.count,
			LastCompletedAt: acc.lastCompletedAt,
			LastCompletedBy: acc.lastCompletedBy,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].LastCompletedAt == nil {
			return false
		}
		if entries[j].LastCompletedAt == nil {
			return true
		}
		return entries[i].LastCompletedAt.After(*entries[j].LastCompletedAt)
	})

	return entries
}

func (handler *ChoreHandler) CreateForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	categories, err := handler.categoryRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding categories", "error", err)
	}

	users, err := handler.userRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding users", "error", err)
	}

	component := pages.ChoreForm(pages.ChoreFormProps{
		User:       user,
		Categories: categories,
		AllUsers:   users,
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

	recurrenceType := models.RecurrenceType(r.FormValue("recurrence_type"))
	recurrenceValue := buildRecurrenceValue(recurrenceType, r)

	chore := models.Chore{
		Name:            r.FormValue("name"),
		Description:     r.FormValue("description"),
		CreatedByUserID: user.ID,
		RecurrenceType:  recurrenceType,
		RecurrenceValue: recurrenceValue,
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

	if assignees := r.Form["assignees"]; len(assignees) > 0 {
		if err := handler.choreRepo.SetEligibleAssignees(ctx, created.ID, assignees); err != nil {
			slog.Error("setting eligible assignees", "error", err)
		}
	}

	assigned, err := handler.choreService.AssignNextUser(ctx, created)
	if err != nil {
		slog.Error("assigning chore", "error", err)
	}

	if created.RecurrenceType != models.RecurrenceNone && !created.RecurOnComplete {
		seriesID := created.ID
		assigned.SeriesID = &seriesID
		if err := handler.choreRepo.Update(ctx, assigned); err != nil {
			slog.Error("setting series_id on new chore", "error", err)
		} else {
			if err := handler.choreService.SeedFutureOccurrences(ctx, assigned, time.Now().AddDate(1, 0, 0)); err != nil {
				slog.Error("seeding future occurrences for new chore", "error", err)
			}
		}
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

	users, err := handler.userRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding users", "error", err)
	}

	eligibleAssignees, err := handler.choreRepo.GetEligibleAssignees(ctx, choreID)
	if err != nil {
		slog.Error("getting eligible assignees", "error", err)
	}
	chore.EligibleAssignees = eligibleAssignees

	component := pages.ChoreForm(pages.ChoreFormProps{
		User:       user,
		Categories: categories,
		AllUsers:   users,
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

	// Capture old recurrence settings before overwriting
	oldRecurrenceType := chore.RecurrenceType
	oldRecurrenceValue := chore.RecurrenceValue

	recurrenceType := models.RecurrenceType(r.FormValue("recurrence_type"))
	recurrenceValue := buildRecurrenceValue(recurrenceType, r)

	chore.Name = r.FormValue("name")
	chore.Description = r.FormValue("description")
	chore.RecurrenceType = recurrenceType
	chore.RecurrenceValue = recurrenceValue
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

	if assignees := r.Form["assignees"]; len(assignees) > 0 {
		if err := handler.choreRepo.SetEligibleAssignees(ctx, chore.ID, assignees); err != nil {
			slog.Error("setting eligible assignees", "error", err)
		}
	} else {
		if err := handler.choreRepo.SetEligibleAssignees(ctx, chore.ID, nil); err != nil {
			slog.Error("clearing eligible assignees", "error", err)
		}
	}

	// If recurrence changed, delete stale future instances and re-seed
	recurrenceChanged := recurrenceType != oldRecurrenceType || recurrenceValue != oldRecurrenceValue
	if recurrenceChanged && chore.SeriesID != nil && !chore.RecurOnComplete {
		if err := handler.choreRepo.DeleteFuturePendingBySeries(ctx, *chore.SeriesID); err != nil {
			slog.Error("deleting stale future instances", "error", err)
		} else if err := handler.choreService.SeedFutureOccurrences(ctx, chore, time.Now().AddDate(1, 0, 0)); err != nil {
			slog.Error("re-seeding after recurrence change", "error", err)
		}
	}

	http.Redirect(w, r, "/chores", http.StatusFound)
}

func (handler *ChoreHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	choreID := chi.URLParam(r, "id")

	chore, err := handler.choreRepo.FindByID(ctx, choreID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Delete future pending siblings before deleting the chore itself
	if chore.SeriesID != nil {
		if err := handler.choreRepo.DeleteFuturePendingBySeries(ctx, *chore.SeriesID); err != nil {
			slog.Error("deleting future pending siblings", "error", err)
		}
	}

	if err := handler.choreRepo.Delete(ctx, choreID); err != nil {
		slog.Error("deleting chore", "error", err)
		http.Error(w, "Error deleting chore", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/chores", http.StatusFound)
}

func (handler *ChoreHandler) DeleteHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if err := handler.choreRepo.DeleteCompletedByName(ctx, name); err != nil {
		slog.Error("deleting chore history by name", "error", err, "name", name)
		http.Error(w, "Error deleting history", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
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
		users, _ := handler.userRepo.FindAll(ctx)
		userNameMap := make(map[string]string, len(users))
		userAvatarMap := make(map[string]string, len(users))
		for _, u := range users {
			userNameMap[u.ID] = u.Name
			userAvatarMap[u.ID] = u.AvatarURL
		}
		component := pages.ChoreRow(chore, user, userNameMap, userAvatarMap)
		component.Render(ctx, w)
		return
	}

	http.Redirect(w, r, "/chores", http.StatusFound)
}

func (handler *ChoreHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	choreID := chi.URLParam(r, "id")

	chore, err := handler.choreRepo.FindByID(ctx, choreID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var assignedToName, assignedToAvatar string
	if chore.AssignedToUserID != nil {
		assignedUser, err := handler.userRepo.FindByID(ctx, *chore.AssignedToUserID)
		if err == nil {
			assignedToName = assignedUser.Name
			assignedToAvatar = assignedUser.AvatarURL
		}
	}

	var categoryName string
	if chore.CategoryID != nil {
		category, err := handler.categoryRepo.FindByID(ctx, *chore.CategoryID)
		if err == nil {
			categoryName = category.Name
		}
	}

	component := pages.ChoreDetailFragment(chore, assignedToName, assignedToAvatar, categoryName)
	component.Render(ctx, w)
}

type recurrenceConfigJSON struct {
	Interval   int      `json:"interval,omitempty"`
	Unit       string   `json:"unit,omitempty"`
	Days       []string `json:"days,omitempty"`
	DayOfMonth int      `json:"day_of_month,omitempty"`
}

func buildRecurrenceValue(recurrenceType models.RecurrenceType, r *http.Request) string {
	if recurrenceType == models.RecurrenceNone || recurrenceType == models.RecurrenceDaily {
		return ""
	}

	config := recurrenceConfigJSON{}

	if intervalStr := r.FormValue("recurrence_interval"); intervalStr != "" {
		if interval, err := strconv.Atoi(intervalStr); err == nil && interval > 0 {
			config.Interval = interval
		}
	}
	if config.Interval == 0 {
		config.Interval = 1
	}

	switch recurrenceType {
	case models.RecurrenceWeekly:
		config.Days = r.Form["recurrence_days"]
	case models.RecurrenceMonthly:
		if dayStr := r.FormValue("recurrence_day_of_month"); dayStr != "" {
			if day, err := strconv.Atoi(dayStr); err == nil && day >= 1 && day <= 31 {
				config.DayOfMonth = day
			}
		}
	case models.RecurrenceCustom:
		config.Unit = r.FormValue("recurrence_unit")
		if config.Unit == "" {
			config.Unit = "days"
		}
	}

	jsonBytes, err := json.Marshal(config)
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}
