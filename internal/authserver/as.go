// Package authserver is a simple HTTP server for exchanging code to tokens.
package authserver

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"unsafe"

	"go.uber.org/zap"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func StartServer(cID, cSecret string, log *zap.Logger) {
	const op = "authserver.initServer"

	verifier, challenge := generatePKCEPair()
	authURL := fmt.Sprintf(
		"https://id.twitch.tv/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=chat:read+chat:edit&code_challenge=%s&code_challenge_method=S256",
		cID,
		"http://localhost:51236",
		challenge,
	)

	log.Debug("Auth URL",
		zap.String("op", op),
		zap.String("authURL", authURL))

	err := exec.Command("xdg-open", authURL).Start()
	if err != nil {
		log.Error("error opening browser. Use -d flag to see URL",
			zap.String("op", op),
			zap.Error(err))
	}

	log.Info("Auth mode enabled")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing code", http.StatusBadRequest)
			log.Error("Missing code",
				zap.String("op", op),
				zap.Any("Request body", r.Body))
			return
		}

		log.Debug("Code",
			zap.String("op", op),
			zap.String("code", code))

		tokens, err := exchangeCode(code, verifier, cID, cSecret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Error("Error exchanging code",
				zap.String("op", op),
				zap.String("code", code),
				zap.Error(err))
			return
		}

		log.Debug("Tokens",
			zap.String("op", op),
			zap.String("accessToken", tokens.AccessToken),
			zap.String("refreshToken", tokens.RefreshToken),
			zap.Int("expiresIn", tokens.ExpiresIn))

		if err := writeData(tokens, cID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Error("Error writing data",
				zap.String("op", op),
				zap.Error(err))
			return
		}

		log.Debug("Data written",
			zap.String("op", op))

		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Addr:    ":51236",
		Handler: mux,
	}

	log.Info("Starting HTTP server",
		zap.String("op", op),
		zap.String("addr", srv.Addr))

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal("HTTP server error",
			zap.String("op", op),
			zap.Error(err))
	}
}

func generatePKCEPair() (string, string) {
	b := make([]byte, 32)
	rand.Read(b)
	verifier := base64.RawURLEncoding.EncodeToString(b)

	verBytes := unsafe.Slice(unsafe.StringData(verifier), len(verifier))
	hash := sha256.Sum256(verBytes)
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return verifier, challenge
}

func exchangeCode(code, verifier, cID, cSecret string) (*TokenResponse, error) {
	const op = "authserver.exchangeCode"

	data := url.Values{}
	data.Set("client_id", cID)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", "http://localhost:51236")
	data.Set("code_verifier", verifier)
	data.Set("code_challenge_method", "S256")
	data.Set("client_secret", cSecret)

	req, err := http.NewRequest("POST", "https://id.twitch.tv/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: error reading response body: %w", op, err)
	}

	var result TokenResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("%s: error decoding response: %w", op, err)
	}
	return &result, nil
}

func getUsername(accessToken, cID string) (string, error) {
	const op = "authserver.getUsername"

	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/users", nil)
	if err != nil {
		return "", fmt.Errorf("%s: error creating request: %w", op, err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Client-Id", cID)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: error requesting username: %w", op, err)
	}
	defer res.Body.Close()

	var result struct {
		Data []struct {
			Login       string `json:"login"`
			DisplayName string `json:"display_name"`
			ID          string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("%s: error decoding response: %v", op, err)
	}

	if len(result.Data) == 0 {
		return "", fmt.Errorf("%s: no data in response", op)
	}

	return result.Data[0].DisplayName, nil
}

func writeData(tokens *TokenResponse, cID string) error {
	const op = "authserver.writeData"

	username, errUserName := getUsername(tokens.AccessToken, cID)

	data := fmt.Sprintf("[PrettyChatConfig]\nPASS:%s\nNICK:%s\nJOIN:\n[\\PrettyChatConfig]\n",
		tokens.AccessToken, username)
	dataBytes := unsafe.Slice(unsafe.StringData(data), len(data))

	f, err := os.OpenFile("prcht.gurlf", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("%s: error opening file: %v", op, err)
	}
	defer f.Close()

	if _, err := f.Write(dataBytes); err != nil {
		return fmt.Errorf("%s: error writing to file: %v", op, err)
	}

	if errUserName != nil {
		return fmt.Errorf("%s: error getting username: %v", op, errUserName)
	}

	return nil
}
