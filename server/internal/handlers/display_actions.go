package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/go-chi/chi/v5"
)

type displayPerson struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (handler *APIHandler) ListDisplayPeople(writer http.ResponseWriter, request *http.Request) {
	users, err := handler.userRepo.FindAll(request.Context())
	if err != nil {
		writeAPIError(writer, request, http.StatusInternalServerError, "could not load household members", err)
		return
	}
	people := make([]displayPerson, 0, len(users))
	for _, user := range users {
		people = append(people, displayPerson{ID: user.ID, Name: user.Name})
	}
	writeJSON(writer, http.StatusOK, people)
}

func (handler *APIHandler) CompleteDisplayChore(writer http.ResponseWriter, request *http.Request) {
	var body struct {
		UserID string `json:"userID"`
	}
	if !decodeJSONBody(writer, request, &body) {
		return
	}
	if body.UserID == "" {
		writeJSONError(writer, http.StatusBadRequest, "a household member is required")
		return
	}
	if _, err := handler.userRepo.FindByID(request.Context(), body.UserID); errors.Is(err, sql.ErrNoRows) {
		writeJSONError(writer, http.StatusBadRequest, "choose a current household member")
		return
	} else if err != nil {
		writeAPIError(writer, request, http.StatusInternalServerError, "could not verify household member", err)
		return
	}
	err := handler.choreService.CompleteChore(request.Context(), chi.URLParam(request, "id"), body.UserID)
	switch {
	case err == nil:
		writer.WriteHeader(http.StatusNoContent)
	case errors.Is(err, services.ErrChoreAlreadyComplete):
		writeJSONError(writer, http.StatusConflict, "chore is already complete")
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(writer, http.StatusNotFound, "chore not found")
	default:
		writeAPIError(writer, request, http.StatusInternalServerError, "failed to complete chore", err)
	}
}

func (handler *APIHandler) SaveDisplayMeal(writer http.ResponseWriter, request *http.Request) {
	var body struct {
		Date     string `json:"date"`
		MealType string `json:"mealType"`
		Name     string `json:"name"`
		RecipeID string `json:"recipeID"`
	}
	if !decodeJSONBody(writer, request, &body) {
		return
	}
	meal := models.MealPlan{Date: body.Date, MealType: models.MealType(body.MealType), Name: strings.TrimSpace(body.Name)}
	if body.RecipeID != "" {
		meal.RecipeID = &body.RecipeID
	}
	if meal.Date == "" || meal.Name == "" {
		writeJSONError(writer, http.StatusBadRequest, "date, mealType, and name are required")
		return
	}
	if meal.MealType != models.MealTypeBreakfast && meal.MealType != models.MealTypeLunch && meal.MealType != models.MealTypeDinner {
		writeJSONError(writer, http.StatusBadRequest, "mealType must be breakfast, lunch, or dinner")
		return
	}
	if err := handler.mealPlanRepo.Upsert(request.Context(), meal); err != nil {
		writeAPIError(writer, request, http.StatusInternalServerError, "failed to save meal", err)
		return
	}
	saved, err := handler.mealPlanRepo.FindByDateAndType(request.Context(), meal.Date, meal.MealType)
	if err != nil {
		writeAPIError(writer, request, http.StatusInternalServerError, "failed to retrieve saved meal", err)
		return
	}
	writeJSON(writer, http.StatusOK, saved)
}
