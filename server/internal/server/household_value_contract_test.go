package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestHomeValueContract(t *testing.T) {
	fixture := newContractFixture(t)
	initial := fixture.request(t, http.MethodPatch, "/api/home-info", "admin", `{"latitude":51.5,"longitude":-0.1,"timezone":"Europe/London","arrivalMarginMinutes":10}`, true)
	if initial.Code != http.StatusOK {
		t.Fatalf("initial home: %d %s", initial.Code, initial.Body.String())
	}
	updated := fixture.request(t, http.MethodPatch, "/api/home-info", "admin", `{"arrivalMarginMinutes":25}`, true)
	if updated.Code != http.StatusOK {
		t.Fatalf("partial update: %d %s", updated.Code, updated.Body.String())
	}
	assertHome := func(body []byte, familyName bool) {
		t.Helper()
		var value map[string]any
		if err := json.Unmarshal(body, &value); err != nil {
			t.Fatal(err)
		}
		want := 4
		if familyName {
			want++
			if value["familyName"] != "Family" {
				t.Fatalf("family name: %s", body)
			}
		}
		if len(value) != want || value["latitude"] != 51.5 || value["longitude"] != -0.1 || value["timezone"] != "Europe/London" || value["arrivalMarginMinutes"] != float64(25) {
			t.Fatalf("home fields: %s", body)
		}
	}
	assertHome(updated.Body.Bytes(), false)
	for _, endpoint := range []struct{ path, token string }{{"/api/home-info", "member"}, {"/api/display/home-info", "display"}} {
		response := fixture.request(t, http.MethodGet, endpoint.path, endpoint.token, "", true)
		if response.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", endpoint.path, response.Code, response.Body.String())
		}
		assertHome(response.Body.Bytes(), true)
	}
	invalid := fixture.request(t, http.MethodPatch, "/api/home-info", "admin", `{"latitude":91}`, false)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid home: %d %s", invalid.Code, invalid.Body.String())
	}
	forbidden := fixture.request(t, http.MethodPatch, "/api/home-info", "member", `{"arrivalMarginMinutes":30}`, false)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("member update: %d %s", forbidden.Code, forbidden.Body.String())
	}
	response := fixture.request(t, http.MethodGet, "/api/home-info", "member", "", true)
	assertHome(response.Body.Bytes(), true)
}

func TestScratchpadValueContract(t *testing.T) {
	fixture := newContractFixture(t)
	type drawing struct {
		Strokes []struct {
			Color  string `json:"color"`
			Points []struct {
				X float64 `json:"x"`
				Y float64 `json:"y"`
			} `json:"points"`
		} `json:"strokes"`
		UpdatedAt string `json:"updatedAt"`
	}
	for _, endpoint := range []struct{ path, token string }{{"/api/scratchpad", "member"}, {"/api/display/scratchpad", "display"}} {
		response := fixture.request(t, http.MethodGet, endpoint.path, endpoint.token, "", true)
		var value drawing
		if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil || response.Code != http.StatusOK || value.Strokes == nil || len(value.Strokes) != 0 || value.UpdatedAt == "" {
			t.Fatalf("initial %s: %d %s (%v)", endpoint.path, response.Code, response.Body.String(), err)
		}
	}
	saved := fixture.request(t, http.MethodPut, "/api/display/scratchpad", "display", `{"strokes":[{"color":"#3c654c","points":[{"x":0.2,"y":0.3}]}]}`, true)
	if saved.Code != http.StatusInternalServerError {
		t.Fatalf("display save: %d %s", saved.Code, saved.Body.String())
	}
	saved = fixture.request(t, http.MethodPut, "/api/scratchpad", "member", `{"strokes":[{"color":"#3c654c","points":[{"x":0.2,"y":0.3}]}]}`, true)
	if saved.Code != http.StatusOK {
		t.Fatalf("member save: %d %s", saved.Code, saved.Body.String())
	}
	invalid := fixture.request(t, http.MethodPut, "/api/scratchpad", "member", `{"strokes":[{"color":"#3c654c","points":[{"x":2,"y":0.3}]}]}`, false)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid drawing: %d %s", invalid.Code, invalid.Body.String())
	}
	read := fixture.request(t, http.MethodGet, "/api/scratchpad", "member", "", true)
	var value drawing
	if err := json.Unmarshal(read.Body.Bytes(), &value); err != nil || read.Code != http.StatusOK || len(value.Strokes) != 1 || value.Strokes[0].Color != "#3c654c" || len(value.Strokes[0].Points) != 1 || value.Strokes[0].Points[0].X != 0.2 || value.Strokes[0].Points[0].Y != 0.3 || value.UpdatedAt == "" {
		t.Fatalf("saved drawing: %d %s (%v)", read.Code, read.Body.String(), err)
	}
	cleared := fixture.request(t, http.MethodPut, "/api/scratchpad", "member", `{"strokes":[]}`, true)
	if cleared.Code != http.StatusOK {
		t.Fatalf("clear: %d %s", cleared.Code, cleared.Body.String())
	}
	read = fixture.request(t, http.MethodGet, "/api/display/scratchpad", "display", "", true)
	if err := json.Unmarshal(read.Body.Bytes(), &value); err != nil || value.Strokes == nil || len(value.Strokes) != 0 {
		t.Fatalf("cleared drawing: %s (%v)", read.Body.String(), err)
	}
}
