package handlers

import (
	"context"
	"encoding/json"
	"github.com/bensuskins/family-hub/internal/middleware"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func TestInventoryStock_AdjustsCountAndPreservesDetails(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	repo := repository.NewInventoryRepository(database)
	user := newInventoryTestUser(t, repository.NewUserRepository(database))
	area, err := repo.CreateArea(context.Background(), models.InventoryArea{Name: "Cupboard", CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	item, err := repo.CreateItem(context.Background(), models.InventoryItem{AreaID: area.ID, Name: "Pods", Quantity: 12, Unit: "pods", LowAt: 3, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, repo, nil, nil, "", "", "")
	router := chi.NewRouter()
	router.Patch("/api/inventory/items/{id}/stock", handler.UpdateInventoryStock)
	recorder := doRequest(t, router, http.MethodPatch, "/api/inventory/items/"+item.ID+"/stock", `{"quantityDelta":-2}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status %d: %s", recorder.Code, recorder.Body.String())
	}
	var saved models.InventoryItem
	if err := json.NewDecoder(recorder.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.Quantity != 10 || saved.Name != "Pods" || saved.Unit != "pods" || saved.LowAt != 3 || saved.AreaID != area.ID {
		t.Fatalf("unexpected saved item: %+v", saved)
	}
}

func TestInventoryStock_ValidationAndConcurrentChanges(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	repo := repository.NewInventoryRepository(database)
	user := newInventoryTestUser(t, repository.NewUserRepository(database))
	area, err := repo.CreateArea(context.Background(), models.InventoryArea{Name: "Kitchen", CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	item, err := repo.CreateItem(context.Background(), models.InventoryItem{AreaID: area.ID, Name: "Tins", Quantity: 5, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, repo, nil, nil, "", "", "")
	router := chi.NewRouter()
	router.Patch("/api/inventory/items/{id}/stock", handler.UpdateInventoryStock)
	path := "/api/inventory/items/" + item.ID + "/stock"
	for _, body := range []string{`{}`, `{"quantity":1,"quantityDelta":2}`, `{"quantity":-1}`, `{"quantity":1.5}`, `{"level":20}`, `{"quantity":9223372036854775808}`} {
		recorder := doRequest(t, router, http.MethodPatch, path, body)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: got %d", body, recorder.Code)
		}
	}
	recorder := doRequest(t, router, http.MethodPatch, path, `{"quantityDelta":-12}`)
	if recorder.Code != http.StatusOK {
		t.Fatal(recorder.Body.String())
	}
	var saved models.InventoryItem
	if err := json.Unmarshal(recorder.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Quantity != 0 {
		t.Fatalf("quantity = %d", saved.Quantity)
	}
	errors := make(chan int, 20)
	for range 20 {
		go func() {
			request := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(`{"quantityDelta":1}`))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			errors <- response.Code
		}()
	}
	for range 20 {
		if code := <-errors; code != http.StatusOK {
			t.Fatalf("concurrent response = %d", code)
		}
	}
	saved, err = repo.FindItemByID(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Quantity != 20 {
		t.Fatalf("lost count changes: %d", saved.Quantity)
	}
	if code := doRequest(t, router, http.MethodPatch, "/api/inventory/items/missing/stock", `{"quantity":1}`).Code; code != http.StatusNotFound {
		t.Fatalf("missing item: %d", code)
	}
	level, err := repo.CreateItem(context.Background(), models.InventoryItem{AreaID: area.ID, Name: "Soap", TrackingMode: models.TrackingModeLevel, Level: 50, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	path = "/api/inventory/items/" + level.ID + "/stock"
	for _, body := range []string{`{"quantityDelta":1}`, `{"level":-1}`, `{"level":101}`} {
		if code := doRequest(t, router, http.MethodPatch, path, body).Code; code != http.StatusBadRequest {
			t.Fatalf("%s: %d", body, code)
		}
	}
	recorder = doRequest(t, router, http.MethodPatch, path, `{"level":0}`)
	if recorder.Code != http.StatusOK {
		t.Fatal(recorder.Body.String())
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Level != 0 {
		t.Fatalf("level = %d", saved.Level)
	}
}

func TestInventoryStockSwiftContract(t *testing.T) {
	if os.Getenv("RUN_SWIFT_STOCK_CONTRACT") != "1" {
		t.Skip("Set RUN_SWIFT_STOCK_CONTRACT=1 to test the Swift client against the Go API.")
	}
	database := testutil.NewTestDatabase(t)
	repo := repository.NewInventoryRepository(database)
	user := newInventoryTestUser(t, repository.NewUserRepository(database))
	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, repo, nil, nil, "", "", "")
	var uncertainRequests atomic.Int32
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer siri-contract-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), middleware.UserContextKey, user)))
		})
	})
	router.Get("/api/inventory", handler.ListInventory)
	router.Post("/api/inventory/areas", handler.CreateInventoryArea)
	router.Post("/api/inventory/areas/{id}/items", handler.CreateInventoryItem)
	router.Patch("/api/inventory/items/{id}/stock", func(w http.ResponseWriter, r *http.Request) {
		if chi.URLParam(r, "id") == "uncertain" {
			uncertainRequests.Add(1)
			writeJSONError(w, http.StatusServiceUnavailable, "write not confirmed")
			return
		}
		handler.UpdateInventoryStock(w, r)
	})
	fixture := httptest.NewServer(router)
	defer fixture.Close()
	packagePath, err := filepath.Abs("../../../ios/FamilyHubKit")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "swift", "test", "--package-path", packagePath, "--scratch-path", filepath.Join(t.TempDir(), "swift"), "--jobs", "4", "--filter", "SiriStockHTTPContractTests")
	command.Env = append(os.Environ(), "FAMILY_HUB_STOCK_CONTRACT_URL="+fixture.URL)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("Swift stock contract: %v\n%s", err, output)
	}
	if uncertainRequests.Load() != 1 {
		t.Fatalf("relative stock write retried %d times", uncertainRequests.Load())
	}
}

func TestInventoryStock_RejectsCountOverflow(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	repo := repository.NewInventoryRepository(database)
	user := newInventoryTestUser(t, repository.NewUserRepository(database))
	area, err := repo.CreateArea(context.Background(), models.InventoryArea{Name: "Kitchen", CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	item, err := repo.CreateItem(context.Background(), models.InventoryItem{AreaID: area.ID, Name: "Tins", Quantity: math.MaxInt, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, repo, nil, nil, "", "", "")
	router := chi.NewRouter()
	router.Patch("/api/inventory/items/{id}/stock", handler.UpdateInventoryStock)
	response := doRequest(t, router, http.MethodPatch, "/api/inventory/items/"+item.ID+"/stock", `{"quantityDelta":1}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("overflow response: %d: %s", response.Code, response.Body.String())
	}
	saved, err := repo.FindItemByID(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Quantity != math.MaxInt {
		t.Fatalf("quantity changed after overflow: %d", saved.Quantity)
	}
}
