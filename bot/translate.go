package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ---------------------- HELPER FUNCTIONS FOR TRANSLATION ----------------------

// Map from BBB to Libretranslate language codes
var bbbToLibretranslate = map[string]string{
	"ar":    "ar",
	"az":    "az",
	"bg-BG": "bg",
	"bn":    "bn",
	"ca":    "ca",
	"cs-CZ": "cs",
	"da":    "da",
	"de":    "de",
	"el-GR": "el",
	"en":    "en",
	"eo":    "eo",
	"es":    "es",
	"es-419": "es",
	"es-ES": "es",
	"es-MX": "es",
	"et":    "et",
	"fa-IR": "fa",
	"fi":    "fi",
	"fr":    "fr",
	"he":    "he",
	"hi-IN": "hi",
	"hu-HU": "hu",
	"id":    "id",
	"it-IT": "it",
	"ja":    "ja",
	"ko-KR": "ko",
	"lt-LT": "lt",
	"lv":    "lv",
	"nb-NO": "nb",
	"nl":    "nl",
	"pl-PL": "pl",
	"pt":    "pt",
	"pt-BR": "pt",
	"ro-RO": "ro",
	"ru":    "ru",
	"sk-SK": "sk",
	"sl":    "sl",
	"sv-SE": "sv",
	"th":    "th",
	"tr-TR": "tr",
	"uk-UA": "uk",
	"zh-CN": "zh",
}

// ConvertBBBToLibretranslate converts BBB language code to Libretranslate language code
func ConvertBBBToLibretranslate(bbbCode string) string {
	if code, exists := bbbToLibretranslate[bbbCode]; exists {
		return code
	}
	return ""
}

// TranslationRequest struct to hold the request payload for translation
type TranslationRequest struct {
	Q      string `json:"q"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// TranslationResponse struct to parse the response
type TranslationResponse struct {
	TranslatedText string `json:"translatedText"`
}

// translate function sends a request to the LibreTranslate API and returns the translated text
func translate(apiURL, text, sourceLang, targetLang string) (string, error) {
	linreTargetLang := ConvertBBBToLibretranslate(targetLang)
	if linreTargetLang == "" {
		return "", fmt.Errorf("unsupported language: %s", targetLang)
	}

	// Create the request payload
	requestPayload := TranslationRequest{
		Q:      text,
		Source: sourceLang,
		Target: linreTargetLang,
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("error marshalling request payload: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("error creating HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get a valid response. Status code: %d, Response: %s", resp.StatusCode, body)
	}

	var translationResponse TranslationResponse
	err = json.Unmarshal(body, &translationResponse)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}

	return translationResponse.TranslatedText, nil
}

// getSupportedLanguages returns a list of supported languages
func getSupportedLanguages() []string {
	languages := []string{}
	for key := range bbbToLibretranslate {
		if bbbToLibretranslate[key] != "" {
			languages = append(languages, key)
		}
	}
	return languages
}