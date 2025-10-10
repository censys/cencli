package config

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// setViperDefaults automatically sets viper values from a config struct using yaml tags
func setViperDefaults(cfg *Config) cenclierrors.CencliError {
	return setViperDefaultsHelper(reflect.ValueOf(cfg).Elem(), reflect.TypeOf(cfg).Elem(), "")
}

// setViperDefaultsHelper handles nested structs, slices, and maps
func setViperDefaultsHelper(v reflect.Value, t reflect.Type, prefix string) cenclierrors.CencliError {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Get the yaml tag to use as the viper key
		yamlTag := fieldType.Tag.Get("yaml")
		if yamlTag == "" || yamlTag == "-" {
			continue
		}

		// Build the full key path for nested structures
		var key string
		if prefix == "" {
			key = yamlTag
		} else {
			key = prefix + "." + yamlTag
		}

		// Handle the field value based on its type
		if err := setViperValue(field, key); err != nil {
			return newInvalidConfigErrorWithKey(key, err.Error())
		}
	}

	return nil
}

// setViperValue sets a viper value based on the field's type and value
func setViperValue(field reflect.Value, key string) error {
	// Handle time.Duration specially before checking Kind() since it has underlying type int64
	if duration, ok := field.Interface().(time.Duration); ok {
		viper.SetDefault(key, duration.String())
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		viper.SetDefault(key, field.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		viper.SetDefault(key, field.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		viper.SetDefault(key, field.Uint())
	case reflect.Bool:
		viper.SetDefault(key, field.Bool())
	case reflect.Float32, reflect.Float64:
		viper.SetDefault(key, field.Float())
	case reflect.Slice, reflect.Array:
		// Handle slices and arrays - this ensures empty slices show as [] in YAML
		slice := make([]interface{}, field.Len())
		for i := 0; i < field.Len(); i++ {
			slice[i] = field.Index(i).Interface()
		}
		viper.SetDefault(key, slice)
	case reflect.Map:
		// Handle maps - this ensures empty maps show as {} in YAML
		if field.IsNil() {
			viper.SetDefault(key, map[string]interface{}{})
		} else {
			mapValue := make(map[string]interface{})
			for _, mapKey := range field.MapKeys() {
				mapValue[fmt.Sprintf("%v", mapKey.Interface())] = field.MapIndex(mapKey).Interface()
			}
			// Always set the map, even if empty, to ensure it shows in YAML
			viper.SetDefault(key, mapValue)
		}
	case reflect.Struct:
		// Handle nested structs recursively
		return setViperDefaultsHelper(field, field.Type(), key)
	case reflect.Ptr:
		// Handle pointers
		if !field.IsNil() {
			return setViperValue(field.Elem(), key)
		}
		viper.SetDefault(key, nil)
	default:
		// For custom types, check if they have a String() method
		if stringer, ok := field.Interface().(interface{ String() string }); ok {
			viper.SetDefault(key, stringer.String())
		} else {
			viper.SetDefault(key, field.Interface())
		}
	}

	return nil
}

// addDocCommentsToYAML reads a YAML file and adds inline comments based on doc struct tags.
// It finds lines matching YAML keys and appends comments from the corresponding doc tags.
func addDocCommentsToYAML(yamlPath string, cfg *Config) error {
	docMap := make(map[string]string)
	buildDocMap(reflect.TypeOf(cfg).Elem(), "", docMap)

	file, err := os.Open(yamlPath)
	if err != nil {
		return fmt.Errorf("failed to open yaml file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read yaml file: %w", err)
	}

	for i, line := range lines {
		lines[i] = addCommentToLine(line, docMap)
	}

	output := strings.Join(lines, "\n")
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	if err := os.WriteFile(yamlPath, []byte(output), 0o644); err != nil {
		return fmt.Errorf("failed to write yaml file: %w", err)
	}

	return nil
}

// buildDocMap recursively builds a map of yaml keys to doc comments
func buildDocMap(t reflect.Type, prefix string, docMap map[string]string) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		yamlTag := field.Tag.Get("yaml")
		if yamlTag == "" || yamlTag == "-" {
			continue
		}

		yamlFieldName := strings.Split(yamlTag, ",")[0]

		var fullKey string
		if prefix == "" {
			fullKey = yamlFieldName
		} else {
			fullKey = prefix + "." + yamlFieldName
		}

		docTag := field.Tag.Get("doc")
		if docTag != "" {
			docMap[fullKey] = docTag
		}

		fieldType := field.Type
		if fieldType.Kind() == reflect.Struct {
			buildDocMap(fieldType, fullKey, docMap)
		}
	}
}

var yamlKeyRegex = regexp.MustCompile(`^(\s*)([a-z0-9_-]+):\s*(.*)$`)

// addCommentToLine adds an inline comment to a YAML line if it matches a key with a doc tag
func addCommentToLine(line string, docMap map[string]string) string {
	if strings.Contains(line, "#") {
		return line
	}

	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return line
	}

	matches := yamlKeyRegex.FindStringSubmatch(line)
	if len(matches) < 4 {
		return line
	}

	indent := matches[1]
	key := matches[2]
	value := matches[3]

	var doc string
	var found bool

	for fullKey, docComment := range docMap {
		if strings.HasSuffix(fullKey, key) || fullKey == key {
			doc = docComment
			found = true
			break
		}
	}

	if !found {
		return line
	}

	if value != "" && value != "{}" && value != "[]" {
		return fmt.Sprintf("%s%s: %s  # %s", indent, key, value, doc)
	}
	return fmt.Sprintf("%s%s:  # %s", indent, key, doc)
}
