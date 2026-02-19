//go:build ignore

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	model  = "google/gemini-3-flash-preview"
	apiURL = "https://openrouter.ai/api/v1/chat/completions"

	systemPrompt = `You are a JSON translator. You will receive a JSON object representing part of a Swagger/OpenAPI specification. The documentation is primarily in Ukrainian, often with English translations included.

Your task:
1. Translate ALL Ukrainian text to English.
2. Where English translations already exist alongside Ukrainian, keep only the clean English version.
3. Preserve the exact JSON structure, keys, types, and all non-text values.
4. Only translate human-readable text fields like "summary", "description", "tags", etc.
5. Do NOT translate technical identifiers, schema names, $ref values, format values, or type values.
6. Do NOT translate any output examples. The API will return Ukrainian - translating this is inaccurate!
6. Respond with ONLY the translated JSON object, no markdown fences, no explanation.`
)

var (
	apiKey     string
	httpClient *http.Client
)

func init() {
	apiKey = os.Getenv("OPENROUTER_KEY")
	httpClient = &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

// orderedMap preserves insertion order of keys.
type orderedMap struct {
	keys   []string
	values map[string]json.RawMessage
}

func newOrderedMap() *orderedMap {
	return &orderedMap{values: make(map[string]json.RawMessage)}
}

func (o *orderedMap) set(key string, val json.RawMessage) {
	if _, exists := o.values[key]; !exists {
		o.keys = append(o.keys, key)
	}
	o.values[key] = val
}

func (o *orderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyJSON, _ := json.Marshal(k)
		buf.Write(keyJSON)
		buf.WriteByte(':')
		buf.Write(o.values[k])
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// parseOrderedKeys extracts the top-level key order from a JSON object.
func parseOrderedKeys(data json.RawMessage) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := t.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("expected object")
	}
	var keys []string
	depth := 0
	for dec.More() {
		t, err = dec.Token()
		if err != nil {
			return nil, err
		}
		if depth == 0 {
			if key, ok := t.(string); ok {
				keys = append(keys, key)
				// skip the value
				var raw json.RawMessage
				if err := dec.Decode(&raw); err != nil {
					return nil, err
				}
				continue
			}
		}
		if delim, ok := t.(json.Delim); ok {
			switch delim {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
		}
	}
	return keys, nil
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func translate(input json.RawMessage) (json.RawMessage, error) {
	payload := chatRequest{
		Model: model,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: string(input)},
		},
		Temperature: 0.1,
	}

	body, _ := json.Marshal(payload)

	for attempt := range 3 {
		req, _ := http.NewRequest("POST", apiURL, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			fmt.Printf("  Attempt %d failed: %v\n", attempt+1, err)
			time.Sleep(time.Duration(1<<attempt) * time.Second)
			continue
		}

		var cr chatResponse
		err = json.NewDecoder(resp.Body).Decode(&cr)
		resp.Body.Close()
		if err != nil || len(cr.Choices) == 0 {
			fmt.Printf("  Attempt %d failed: bad response\n", attempt+1)
			time.Sleep(time.Duration(1<<attempt) * time.Second)
			continue
		}

		content := strings.TrimSpace(cr.Choices[0].Message.Content)
		// Strip markdown fences
		if strings.HasPrefix(content, "```") {
			if idx := strings.Index(content, "\n"); idx != -1 {
				content = content[idx+1:]
			}
			if idx := strings.LastIndex(content, "```"); idx != -1 {
				content = content[:idx]
			}
			content = strings.TrimSpace(content)
		}

		if !json.Valid([]byte(content)) {
			fmt.Printf("  Attempt %d failed: invalid JSON response\n", attempt+1)
			time.Sleep(time.Duration(1<<attempt) * time.Second)
			continue
		}

		return json.RawMessage(content), nil
	}

	return input, fmt.Errorf("failed after 3 attempts")
}

func main() {
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: OPENROUTER_KEY environment variable is required")
		os.Exit(1)
	}

	_, thisFile, _, _ := runtime.Caller(0)
	specDir := filepath.Dir(thisFile)
	swaggerPath := filepath.Join(specDir, "swagger.json")
	outputPath := filepath.Join(specDir, "swagger_en.json")

	data, err := os.ReadFile(swaggerPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading swagger.json: %v\n", err)
		os.Exit(1)
	}

	// Parse top-level key order
	topKeys, err := parseOrderedKeys(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing key order: %v\n", err)
		os.Exit(1)
	}

	var spec map[string]json.RawMessage
	if err := json.Unmarshal(data, &spec); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing swagger.json: %v\n", err)
		os.Exit(1)
	}

	// Parse paths and preserve order
	pathKeys, err := parseOrderedKeys(spec["paths"])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing paths order: %v\n", err)
		os.Exit(1)
	}

	var paths map[string]json.RawMessage
	if err := json.Unmarshal(spec["paths"], &paths); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing paths: %v\n", err)
		os.Exit(1)
	}

	// Translate all endpoints in parallel
	type result struct {
		path string
		data json.RawMessage
	}

	results := make(chan result, len(paths))
	var wg sync.WaitGroup

	total := len(paths)
	var idx int
	for path, methods := range paths {
		idx++
		wg.Go(func() {
			fmt.Printf("[%d/%d] Translating %s...\n", idx, total, path)
			translated, err := translate(methods)
			if err != nil {
				fmt.Printf("  WARNING: %s - %v, using original\n", path, err)
				translated = methods
			}
			results <- result{path: path, data: translated}
		})
	}

	wg.Wait()
	close(results)

	translatedPaths := make(map[string]json.RawMessage)
	for r := range results {
		translatedPaths[r.path] = r.data
	}

	// Rebuild paths in original order
	orderedPaths := newOrderedMap()
	for _, key := range pathKeys {
		if val, ok := translatedPaths[key]; ok {
			orderedPaths.set(key, val)
		}
	}
	pathsJSON, _ := json.Marshal(orderedPaths)
	spec["paths"] = json.RawMessage(pathsJSON)

	// Translate components schemas in parallel batches
	if raw, ok := spec["components"]; ok {
		compKeys, _ := parseOrderedKeys(raw)
		var components map[string]json.RawMessage
		json.Unmarshal(raw, &components)

		if schemasRaw, ok := components["schemas"]; ok {
			schemaKeys, _ := parseOrderedKeys(schemasRaw)
			var schemas map[string]json.RawMessage
			json.Unmarshal(schemasRaw, &schemas)

			type schemaResult struct {
				name string
				data json.RawMessage
			}

			// Build ordered list from original key order
			type schemaItem struct {
				name string
				data json.RawMessage
			}
			schemaItems := make([]schemaItem, 0, len(schemaKeys))
			for _, name := range schemaKeys {
				schemaItems = append(schemaItems, schemaItem{name, schemas[name]})
			}

			schemaResults := make(chan schemaResult, len(schemas))
			var swg sync.WaitGroup

			batchSize := 10
			totalBatches := (len(schemaItems) + batchSize - 1) / batchSize

			for i := 0; i < len(schemaItems); i += batchSize {
				end := min(i+batchSize, len(schemaItems))
				batch := schemaItems[i:end]
				batchNum := i/batchSize + 1

				swg.Go(func() {
					fmt.Printf("  Translating schema batch %d/%d (%d schemas)...\n", batchNum, totalBatches, len(batch))
					batchMap := newOrderedMap()
					for _, s := range batch {
						batchMap.set(s.name, s.data)
					}
					batchJSON, _ := json.Marshal(batchMap)
					translated, err := translate(json.RawMessage(batchJSON))
					if err != nil {
						fmt.Printf("  WARNING: schema batch %d failed, using original\n", batchNum)
						for _, s := range batch {
							schemaResults <- schemaResult{name: s.name, data: s.data}
						}
						return
					}
					var translatedMap map[string]json.RawMessage
					if err := json.Unmarshal(translated, &translatedMap); err != nil {
						for _, s := range batch {
							schemaResults <- schemaResult{name: s.name, data: s.data}
						}
						return
					}
					for name, d := range translatedMap {
						schemaResults <- schemaResult{name: name, data: d}
					}
				})
			}

			swg.Wait()
			close(schemaResults)

			translatedSchemas := make(map[string]json.RawMessage)
			for r := range schemaResults {
				translatedSchemas[r.name] = r.data
			}

			// Rebuild schemas in original order
			orderedSchemas := newOrderedMap()
			for _, key := range schemaKeys {
				if val, ok := translatedSchemas[key]; ok {
					orderedSchemas.set(key, val)
				}
			}
			schemasJSON, _ := json.Marshal(orderedSchemas)
			components["schemas"] = json.RawMessage(schemasJSON)

			// Rebuild components in original order
			orderedComps := newOrderedMap()
			for _, key := range compKeys {
				if val, ok := components[key]; ok {
					orderedComps.set(key, val)
				}
			}
			compsJSON, _ := json.Marshal(orderedComps)
			spec["components"] = json.RawMessage(compsJSON)
		}
	}

	// Translate info
	if raw, ok := spec["info"]; ok {
		fmt.Println("Translating info section...")
		translated, err := translate(raw)
		if err == nil {
			spec["info"] = translated
		}
	}

	// Translate tags
	if raw, ok := spec["tags"]; ok {
		fmt.Println("Translating tags...")
		translated, err := translate(raw)
		if err == nil {
			spec["tags"] = translated
		}
	}

	// Build final output preserving top-level key order
	topLevel := newOrderedMap()
	for _, key := range topKeys {
		if val, ok := spec[key]; ok {
			topLevel.set(key, val)
		}
	}

	compact, _ := json.Marshal(topLevel)
	// Pretty-print with indent
	var buf bytes.Buffer
	json.Indent(&buf, compact, "", "  ")

	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nDone! Translated spec written to %s\n", outputPath)
}
