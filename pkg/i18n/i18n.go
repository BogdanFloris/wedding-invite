package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultLanguage is the fallback language
const DefaultLanguage = "ro"

// translations stores all language translations
var translations = map[string]map[string]interface{}{}

// Initialize loads all translation files
func Initialize() error {
	// Get the base directory
	baseDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Load each language file
	languageFiles := []string{"ro.json", "en.json"}
	localesDir := filepath.Join(baseDir, "locales")

	for _, file := range languageFiles {
		lang := strings.TrimSuffix(file, ".json")
		filePath := filepath.Join(localesDir, file)

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read language file %s: %w", file, err)
		}

		var langData map[string]interface{}
		if err := json.Unmarshal(data, &langData); err != nil {
			return fmt.Errorf("failed to parse language file %s: %w", file, err)
		}

		translations[lang] = langData
	}

	return nil
}

// T returns a translation for a key in the specified language
// It supports nested keys with dot notation, e.g., "wedding.title"
func T(lang, key string) string {
	// Default to Romanian if language not found
	if _, ok := translations[lang]; !ok {
		lang = DefaultLanguage
	}

	// Split the key by dots to navigate nested structure
	parts := strings.Split(key, ".")

	// Start with the full translation map for the language
	var current interface{} = translations[lang]

	// Navigate through the nested structure
	for _, part := range parts {
		// Try to cast current to a map
		if currentMap, ok := current.(map[string]interface{}); ok {
			// Look for the next part in the current map
			if val, exists := currentMap[part]; exists {
				current = val
			} else {
				// Key part not found, return the key itself
				return key
			}
		} else {
			// If current is not a map, we can't go deeper
			return key
		}
	}

	// Check if the final value is a string
	if result, ok := current.(string); ok {
		return result
	}

	// If we got here, the value is not a string or doesn't exist
	return key
}

// GetLanguage extracts language from cookie or defaults to Romanian
func GetLanguage(cookieValue string) string {
	if cookieValue == "en" {
		return "en"
	}
	return DefaultLanguage
}

// AvailableLanguages returns a list of available languages
func AvailableLanguages() []string {
	languages := make([]string, 0, len(translations))
	for lang := range translations {
		languages = append(languages, lang)
	}
	return languages
}
