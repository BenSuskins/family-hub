package handlers

import (
	"io"
	"net/http"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
)

const maxBinScheduleBytes = 1 << 20

type BinCollectionsHandler struct {
	settings      repository.SettingsRepository
	pushAvailable bool
}

func NewBinCollectionsHandler(settings repository.SettingsRepository, pushAvailable bool) *BinCollectionsHandler {
	return &BinCollectionsHandler{settings: settings, pushAvailable: pushAvailable}
}

func (handler *BinCollectionsHandler) Get(w http.ResponseWriter, r *http.Request) {
	schedule, err := services.LoadBinSchedule(r.Context(), handler.settings)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load bin schedule", err)
		return
	}
	writeJSON(w, http.StatusOK, binScheduleResponse(schedule, handler.pushAvailable))
}

func (handler *BinCollectionsHandler) GetForDisplay(w http.ResponseWriter, r *http.Request) {
	schedule, err := services.LoadBinSchedule(r.Context(), handler.settings)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load bin schedule", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"collections": schedule.Collections})
}

func (handler *BinCollectionsHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBinScheduleBytes+64*1024)
	if err := r.ParseMultipartForm(maxBinScheduleBytes); err != nil {
		writeJSONError(w, http.StatusBadRequest, "choose a valid ICS calendar file under 1 MB")
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()
	file, _, err := r.FormFile("schedule")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "schedule ICS file is required")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBinScheduleBytes+1))
	if err != nil || len(data) > maxBinScheduleBytes {
		writeJSONError(w, http.StatusBadRequest, "ICS calendar file must be under 1 MB")
		return
	}
	collections, err := services.ParseBinSchedule(string(data), time.Now())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "calendar has no upcoming bin collections or is not a valid ICS file")
		return
	}
	if err := services.SaveBinSchedule(r.Context(), handler.settings, collections, handler.pushAvailable, time.Now()); err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to save bin schedule", err)
		return
	}
	schedule, err := services.LoadBinSchedule(r.Context(), handler.settings)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load saved bin schedule", err)
		return
	}
	writeJSON(w, http.StatusOK, binScheduleResponse(schedule, handler.pushAvailable))
}

func (handler *BinCollectionsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.Enabled && !handler.pushAvailable {
		writeJSONError(w, http.StatusServiceUnavailable, "push notifications are not configured on the server")
		return
	}
	if body.Enabled {
		existing, err := services.LoadBinSchedule(r.Context(), handler.settings)
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to load bin schedule", err)
			return
		}
		if len(existing.Collections) == 0 {
			writeJSONError(w, http.StatusBadRequest, "upload a bin collection calendar before enabling reminders")
			return
		}
	}
	schedule, err := services.UpdateBinScheduleEnabled(r.Context(), handler.settings, body.Enabled)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to update bin reminders", err)
		return
	}
	writeJSON(w, http.StatusOK, binScheduleResponse(schedule, handler.pushAvailable))
}

type binScheduleAPIResponse struct {
	Enabled         bool   `json:"enabled"`
	UploadedAt      string `json:"uploaded_at"`
	CollectionCount int    `json:"collection_count"`
	PushAvailable   bool   `json:"push_available"`
}

func binScheduleResponse(schedule models.BinSchedule, pushAvailable bool) binScheduleAPIResponse {
	return binScheduleAPIResponse{Enabled: schedule.Enabled, UploadedAt: schedule.UploadedAt, CollectionCount: len(schedule.Collections), PushAvailable: pushAvailable}
}
