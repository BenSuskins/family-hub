package handlers

import (
	"errors"
	"net/http"
	"strconv"

	choreapp "github.com/bensuskins/family-hub/internal/application/chores"
	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/go-chi/chi/v5"
)

func (handler *APIHandler) ListChoreDefinitions(w http.ResponseWriter, r *http.Request) {
	result, err := handler.choreService.Definitions(r.Context(), r.URL.Query().Get("include_inactive") == "true")
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load chore definitions", err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (handler *APIHandler) GetChoreDefinition(w http.ResponseWriter, r *http.Request) {
	detail, err := handler.choreService.Definition(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		choreDefinitionError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func choreDefinitionError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, choreapp.ErrChoreNotFound) || errors.Is(err, choreapp.ErrDefinitionNotFound) {
		writeJSONError(w, http.StatusNotFound, "chore definition not found")
	} else {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to change chore definition", err)
	}
}

func (handler *APIHandler) CreateChoreDefinition(w http.ResponseWriter, r *http.Request) {
	var body choreAPIBody
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}
	task := models.Chore{Name: body.Name, Description: body.Description, CreatedByUserID: middleware.GetUser(r.Context()).ID}
	body.applyTo(&task)
	created, err := handler.choreService.CreateChore(r.Context(), task, body.Assignees)
	if err != nil {
		choreDefinitionError(w, r, err)
		return
	}
	detail, err := handler.choreService.Definition(r.Context(), *created.SeriesID)
	if err != nil {
		choreDefinitionError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, detail)
}

func (handler *APIHandler) UpdateChoreDefinition(w http.ResponseWriter, r *http.Request) {
	detail, err := handler.choreService.Definition(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		choreDefinitionError(w, r, err)
		return
	}
	if detail.DeletedAt != nil {
		writeJSONError(w, http.StatusConflict, "chore definition is deleted")
		return
	}
	var body choreAPIBody
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}
	task := detail.Chore
	task.Name = body.Name
	task.Description = body.Description
	body.applyTo(&task)
	eligible := body.Assignees
	if eligible == nil {
		eligible = detail.Chore.EligibleAssignees
	}
	if err := handler.choreService.UpdateDefinition(r.Context(), task, eligible); err != nil {
		choreDefinitionError(w, r, err)
		return
	}
	updated, err := handler.choreService.Definition(r.Context(), detail.ID)
	if err != nil {
		choreDefinitionError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (handler *APIHandler) DeleteChoreDefinition(w http.ResponseWriter, r *http.Request) {
	detail, err := handler.choreService.Definition(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		choreDefinitionError(w, r, err)
		return
	}
	if err := handler.choreService.DeleteSeriesDefinition(r.Context(), detail.ID); err != nil {
		choreDefinitionError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (handler *APIHandler) SkipChore(w http.ResponseWriter, r *http.Request) {
	err := handler.choreService.SkipChore(r.Context(), chi.URLParam(r, "id"), middleware.GetUser(r.Context()).ID)
	if errors.Is(err, choreapp.ErrChoreAlreadyComplete) {
		writeJSONError(w, http.StatusConflict, "chore is no longer outstanding")
		return
	}
	if err != nil {
		choreDefinitionError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (handler *APIHandler) ChoreHistory(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			writeJSONError(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}
		limit = parsed
	}
	history, err := handler.choreService.History(r.Context(), limit, r.URL.Query().Get("cursor"), r.URL.Query().Get("definition_id"))
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load chore history", err)
		return
	}
	writeJSON(w, http.StatusOK, history)
}
