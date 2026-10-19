package services

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

type APNsSender struct {
	key                  *ecdsa.PrivateKey
	keyID, teamID, topic string
	client               *http.Client
	endpoint             func(string) string
	mutex                sync.Mutex
	providerToken        string
	tokenCreated         time.Time
}

func NewAPNsSender(keyID, teamID, topic string, privateKey []byte, client *http.Client) (*APNsSender, error) {
	block, _ := pem.Decode(privateKey)
	if block == nil {
		return nil, errors.New("APNs key must be a PEM private key")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("invalid APNs private key")
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok || key.Curve != elliptic.P256() {
		return nil, errors.New("APNs requires a P-256 private key")
	}
	if keyID == "" || teamID == "" || topic == "" {
		return nil, errors.New("APNs key ID, team ID, and topic are required")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &APNsSender{key: key, keyID: keyID, teamID: teamID, topic: topic, client: client, endpoint: func(environment string) string {
		if environment == "sandbox" {
			return "https://api.sandbox.push.apple.com"
		}
		return "https://api.push.apple.com"
	}}, nil
}

func (sender *APNsSender) authorization(now time.Time) (string, error) {
	sender.mutex.Lock()
	defer sender.mutex.Unlock()
	if sender.providerToken != "" && now.Sub(sender.tokenCreated) < 50*time.Minute {
		return sender.providerToken, nil
	}
	header, _ := json.Marshal(map[string]string{"alg": "ES256", "kid": sender.keyID})
	claims, _ := json.Marshal(map[string]any{"iss": sender.teamID, "iat": now.Unix()})
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(unsigned))
	first, second, err := ecdsa.Sign(rand.Reader, sender.key, digest[:])
	if err != nil {
		return "", errors.New("failed to sign APNs provider token")
	}
	signature := make([]byte, 64)
	first.FillBytes(signature[:32])
	second.FillBytes(signature[32:])
	sender.providerToken = unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
	sender.tokenCreated = now
	return sender.providerToken, nil
}

func (sender *APNsSender) Send(ctx context.Context, device models.NotificationDevice, summary ChoreReminderSummary, deliveryID string, expires time.Time) error {
	return sender.SendMessage(ctx, device, summary.Message(), deliveryID, expires)
}

func (sender *APNsSender) SendMessage(ctx context.Context, device models.NotificationDevice, message PushMessage, deliveryID string, expires time.Time) error {
	if !time.Now().Before(expires) {
		return &PushError{Reason: "notification expired"}
	}
	authorization, err := sender.authorization(time.Now())
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{
		"aps": map[string]any{"alert": map[string]string{"title": message.Title, "body": message.Body}, "sound": "default", "thread-id": message.ThreadID},
		"url": message.URL,
	})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, sender.endpoint(device.Environment)+"/3/device/"+device.Token, bytes.NewReader(payload))
	if err != nil {
		return errors.New("failed to create APNs request")
	}
	request.Header.Set("authorization", "bearer "+authorization)
	request.Header.Set("content-type", "application/json")
	request.Header.Set("apns-topic", sender.topic)
	request.Header.Set("apns-push-type", "alert")
	request.Header.Set("apns-priority", "10")
	request.Header.Set("apns-id", deliveryID)
	request.Header.Set("apns-collapse-id", deliveryID)
	request.Header.Set("apns-expiration", strconv.FormatInt(expires.Unix(), 10))
	response, err := sender.client.Do(request)
	if err != nil {
		return &PushError{Reason: "transport failure", Retry: true}
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return nil
	}
	var failure struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&failure); err != nil {
		failure.Reason = "invalid response"
	}
	invalid := response.StatusCode == 410 || failure.Reason == "BadDeviceToken" || failure.Reason == "DeviceTokenNotForTopic"
	return &PushError{Reason: failure.Reason, InvalidDevice: invalid, Retry: response.StatusCode == 429 || response.StatusCode >= 500}
}
