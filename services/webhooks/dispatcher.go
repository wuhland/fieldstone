package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"
)

const (
	maxConsecutiveFailures = 10
	deliveryTimeout        = 10 * time.Second
)

func sendHTTP(url string, body []byte, sig, eventType string) (int32, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Fieldstone-Signature", "sha256="+sig)
	req.Header.Set("X-Fieldstone-Event", eventType)

	client := &http.Client{Timeout: deliveryTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return int32(resp.StatusCode), nil
}

func computeSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
