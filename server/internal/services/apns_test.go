package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

func TestAPNsSendsSignedAlertAndClassifiesRejections(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	privateKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded})
	status := 200
	reason := ""
	expires := time.Now().Add(5 * time.Minute).Truncate(time.Second)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 {
			t.Error("APNs requires HTTP/2")
		}
		if r.URL.Path != "/3/device/abcdef" || r.Header.Get("apns-topic") != "uk.co.suskins.familyhub" || r.Header.Get("apns-push-type") != "alert" || r.Header.Get("apns-id") != "delivery-id" || r.Header.Get("apns-expiration") != strconv.FormatInt(expires.Unix(), 10) {
			t.Errorf("unexpected APNs request")
		}
		parts := strings.Split(strings.TrimPrefix(r.Header.Get("authorization"), "bearer "), ".")
		if len(parts) != 3 {
			t.Error("missing provider token")
			w.WriteHeader(403)
			return
		}
		signature, _ := base64.RawURLEncoding.DecodeString(parts[2])
		digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
		if len(signature) != 64 || !ecdsa.Verify(&key.PublicKey, digest[:], new(big.Int).SetBytes(signature[:32]), new(big.Int).SetBytes(signature[32:])) {
			t.Error("invalid ES256 signature")
		}
		data, _ := io.ReadAll(r.Body)
		var payload struct {
			APS struct {
				Alert struct {
					Title string `json:"title"`
					Body  string `json:"body"`
				} `json:"alert"`
			} `json:"aps"`
			URL string `json:"url"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			t.Error(err)
		}
		if payload.APS.Alert.Title != "Chore reminder" || !strings.Contains(payload.APS.Alert.Body, "Ben: Empty bins") || payload.URL != "familyhub://today" {
			t.Errorf("payload: %s", data)
		}
		w.WriteHeader(status)
		if status != 200 {
			json.NewEncoder(w).Encode(map[string]string{"reason": reason})
		}
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()
	sender, err := NewAPNsSender("key-id", "team-id", "uk.co.suskins.familyhub", privateKey, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	sender.endpoint = func(string) string { return server.URL }
	device := models.NotificationDevice{Token: "abcdef", Environment: "sandbox"}
	if err := sender.Send(context.Background(), device, ChoreReminderSummary{Count: 3, Tasks: []ChoreReminderTask{{Assignee: "Ben", Name: "Empty bins"}}}, "delivery-id", expires); err != nil {
		t.Fatal(err)
	}
	for _, example := range []struct {
		status         int
		reason         string
		invalid, retry bool
	}{{410, "Unregistered", true, false}, {400, "BadDeviceToken", true, false}, {403, "InvalidProviderToken", false, false}, {429, "TooManyRequests", false, true}, {503, "ServiceUnavailable", false, true}} {
		status, reason = example.status, example.reason
		err := sender.Send(context.Background(), device, ChoreReminderSummary{Count: 3, Tasks: []ChoreReminderTask{{Assignee: "Ben", Name: "Empty bins"}}}, "delivery-id", expires)
		failure, ok := err.(*PushError)
		if !ok || failure.InvalidDevice != example.invalid || failure.Retry != example.retry {
			t.Fatalf("%s: %v", reason, err)
		}
	}
}
