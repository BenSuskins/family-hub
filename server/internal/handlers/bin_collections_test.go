package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func TestBinScheduleUploadRequiresAdmin(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	handler := NewBinCollectionsHandler(repository.NewSettingsRepository(database), true)
	router := binCollectionsRouter(handler, models.User{ID: "member", Role: models.RoleMember})
	request, _ := binScheduleUploadRequest(t, "calendar.ics", testBinScheduleICS)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestBinScheduleUploadReplacesScheduleOnlyAfterValidation(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	settings := repository.NewSettingsRepository(database)
	handler := NewBinCollectionsHandler(settings, true)
	router := binCollectionsRouter(handler, models.User{ID: "admin", Role: models.RoleAdmin})
	request, _ := binScheduleUploadRequest(t, "route.ics", testBinScheduleICS)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	schedule, err := services.LoadBinSchedule(context.Background(), settings)
	if err != nil || len(schedule.Collections) != 1 || schedule.Collections[0].Name != "Food caddy" {
		t.Fatalf("unexpected uploaded schedule: %+v, %v", schedule, err)
	}
	request, _ = binScheduleUploadRequest(t, "bad.ics", "BEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid upload 400, got %d", recorder.Code)
	}
	schedule, err = services.LoadBinSchedule(context.Background(), settings)
	if err != nil || len(schedule.Collections) != 1 || schedule.Collections[0].Name != "Food caddy" {
		t.Fatalf("invalid upload replaced the saved schedule: %+v, %v", schedule, err)
	}
}

const testBinScheduleICS = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:food\r\nDTSTART;VALUE=DATE:20990101\r\nSUMMARY:Food Waste\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

func binScheduleUploadRequest(t *testing.T, filename, contents string) (*http.Request, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("schedule", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte(contents)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/bin-collections", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request, writer.FormDataContentType()
}

func binCollectionsRouter(handler *BinCollectionsHandler, user models.User) *chi.Mux {
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	router.Use(middleware.RequireAdmin)
	router.Post("/api/bin-collections", handler.Upload)
	return router
}

func TestBinScheduleStatusResponseDoesNotExposeCalendar(t *testing.T) {
	response := binScheduleAPIResponse{Enabled: true, UploadedAt: "2026-10-25T12:00:00Z", CollectionCount: 3, PushAvailable: true}
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("collections")) || bytes.Contains(data, []byte("address")) {
		t.Fatalf("status response included private schedule details: %s", data)
	}
}
