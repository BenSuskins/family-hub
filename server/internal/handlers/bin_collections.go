package handlers

import (
	"errors"
	"io"
	"net/http"

	"github.com/bensuskins/family-hub/internal/application/bins"
)

const maxBinScheduleBytes = 1 << 20

type BinCollectionsHandler struct{ service *bins.Service }

func NewBinCollectionsHandler(service *bins.Service) *BinCollectionsHandler {
	return &BinCollectionsHandler{service: service}
}

func (handler *BinCollectionsHandler) Get(w http.ResponseWriter, r *http.Request) {
	schedule, err := handler.service.Get(r.Context())
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load bin schedule", err)
		return
	}
	writeJSON(w, http.StatusOK, binScheduleResponse(schedule, handler.service.PushAvailable()))
}

func (handler *BinCollectionsHandler) GetForDisplay(w http.ResponseWriter, r *http.Request) {
	schedule, err := handler.service.Get(r.Context())
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load bin schedule", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"collections": binCollectionResponses(schedule.Collections)})
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
	schedule, err := handler.service.Upload(r.Context(), string(data))
	if err != nil {
		switch {
		case errors.Is(err, bins.ErrParse):
			writeJSONError(w, http.StatusBadRequest, "calendar has no upcoming bin collections or is not a valid ICS file")
		case errors.Is(err, bins.ErrSave):
			writeAPIError(w, r, http.StatusInternalServerError, "failed to save bin schedule", err)
		default:
			writeAPIError(w, r, http.StatusInternalServerError, "failed to load saved bin schedule", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, binScheduleResponse(schedule, handler.service.PushAvailable()))
}

func (handler *BinCollectionsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if !decodeJSONBody(w, r, &body) {
		return
	}
	schedule, err := handler.service.SetEnabled(r.Context(), body.Enabled)
	if err != nil {
		switch {
		case errors.Is(err, bins.ErrPushUnavailable):
			writeJSONError(w, http.StatusServiceUnavailable, "push notifications are not configured on the server")
		case errors.Is(err, bins.ErrNoCollections):
			writeJSONError(w, http.StatusBadRequest, "upload a bin collection calendar before enabling reminders")
		case errors.Is(err, bins.ErrLoad):
			writeAPIError(w, r, http.StatusInternalServerError, "failed to load bin schedule", err)
		default:
			writeAPIError(w, r, http.StatusInternalServerError, "failed to update bin reminders", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, binScheduleResponse(schedule, handler.service.PushAvailable()))
}

type binScheduleAPIResponse struct {
	Enabled         bool   `json:"enabled"`
	UploadedAt      string `json:"uploaded_at"`
	CollectionCount int    `json:"collection_count"`
	PushAvailable   bool   `json:"push_available"`
}

func binScheduleResponse(schedule bins.Schedule, pushAvailable bool) binScheduleAPIResponse {
	return binScheduleAPIResponse{Enabled: schedule.Enabled, UploadedAt: schedule.UploadedAt, CollectionCount: len(schedule.Collections), PushAvailable: pushAvailable}
}

type binCollectionResponse struct {
	Date string `json:"date"`
	Type string `json:"type"`
	Name string `json:"name"`
}

func binCollectionResponses(collections []bins.Collection) []binCollectionResponse {
	if collections == nil {
		return nil
	}
	result := make([]binCollectionResponse, len(collections))
	for index, collection := range collections {
		result[index] = binCollectionResponse{Date: collection.Date, Type: collection.Type, Name: collection.Name}
	}
	return result
}
