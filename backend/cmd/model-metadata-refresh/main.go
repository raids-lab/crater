// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

const (
	defaultBatchSize          = 100
	maxBatchSize              = 1000
	defaultRequestDelay       = 100 * time.Millisecond
	maxSourceDescriptionBytes = 500
	maxMetadataTags           = 4
	maxStoredReadmeBytes      = 64 * 1024
)

type sourceMetadata struct {
	DisplayName    string
	Description    string
	Readme         string
	License        string
	Task           string
	Library        string
	ModelType      string
	ParameterCount int64
	Private        bool
	Gated          bool
	LoginRequired  bool
	Downloads      int64
	Likes          int64
	SizeBytes      int64
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
	Tags           []string
}

//nolint:gocyclo // Flag-driven batch orchestration is intentionally kept linear for resumability.
func main() {
	apply := flag.Bool("apply", false, "write refreshed metadata to the database")
	batchSize := flag.Int("batch-size", defaultBatchSize, "number of records loaded from the database per batch")
	afterID := flag.Uint("after-id", 0, "resume after this model download ID")
	maxRecords := flag.Int("max-records", 0, "maximum records to process; 0 means unlimited")
	force := flag.Bool("force", false, "refresh records even if their metadata is still fresh")
	staleAfter := flag.Duration("stale-after", 7*24*time.Hour, "refresh metadata older than this duration")
	delay := flag.Duration("delay", defaultRequestDelay, "delay between source API requests")
	source := flag.String("source", "", "optional source filter: huggingface or modelscope")
	flag.Parse()
	if *batchSize < 1 || *batchSize > maxBatchSize {
		panic("batch-size must be between 1 and 1000")
	}
	if *source != "" && *source != string(model.ModelSourceHuggingFace) && *source != string(model.ModelSourceModelScope) {
		panic("source must be huggingface or modelscope")
	}

	db := query.GetDB()
	client := &http.Client{Timeout: 20 * time.Second}
	avatarCache := make(map[string]string)
	var processed, refreshed, failed int
	cursor := *afterID
	stop := false
	for !stop {
		var downloads []model.ModelDownload
		builder := db.Where("status = ? AND id > ?", model.ModelDownloadStatusReady, cursor)
		if *source != "" {
			builder = builder.Where("source = ?", *source)
		}
		if !*force {
			builder = builder.Where("metadata_refreshed_at IS NULL OR metadata_refreshed_at < ?", time.Now().Add(-*staleAfter))
		}
		if err := builder.Order("id ASC").Limit(*batchSize).Find(&downloads).Error; err != nil {
			panic(err)
		}
		if len(downloads) == 0 {
			break
		}

		for i := range downloads {
			download := &downloads[i]
			cursor = download.ID
			processed++
			if err := refreshDownload(db, client, avatarCache, download, *apply); err != nil {
				failed++
				fmt.Printf("FAIL id=%d source=%s name=%s: %v\n", download.ID, download.Source, download.Name, err)
			} else {
				refreshed++
			}
			if *maxRecords > 0 && processed >= *maxRecords {
				stop = true
				break
			}
			if *delay > 0 {
				time.Sleep(*delay)
			}
		}
	}

	fmt.Printf("summary: mode=%s processed=%d refreshed=%d failed=%d next_after_id=%d\n",
		map[bool]string{true: "apply", false: "dry-run"}[*apply], processed, refreshed, failed, cursor)
}

//nolint:gocyclo // One refresh transaction deliberately handles source metadata and the linked dataset together.
func refreshDownload(
	db *gorm.DB,
	client *http.Client,
	avatarCache map[string]string,
	download *model.ModelDownload,
	apply bool,
) error {
	metadata, err := fetchMetadata(client, download)
	if err != nil {
		return err
	}

	organization := strings.SplitN(download.Name, "/", 2)[0]
	logoURL := ""
	if download.Source == model.ModelSourceHuggingFace {
		var ok bool
		logoURL, ok = avatarCache[organization]
		if !ok {
			logoURL, err = fetchHuggingFaceAvatar(client, organization)
			if err != nil {
				return fmt.Errorf("owner avatar lookup: %w", err)
			}
			avatarCache[organization] = logoURL
		}
	}
	sourceURL := repositoryURL(download)
	fmt.Printf("OK   id=%d source=%s name=%s downloads=%d likes=%d tags=%v\n",
		download.ID, download.Source, download.Name, metadata.Downloads, metadata.Likes, metadata.Tags)
	if !apply {
		return nil
	}

	updates := map[string]any{
		"organization":          organization,
		"logo_url":              logoURL,
		"source_url":            sourceURL,
		"display_name":          metadata.DisplayName,
		"source_description":    metadata.Description,
		"source_readme":         metadata.Readme,
		"license":               metadata.License,
		"task":                  metadata.Task,
		"library":               metadata.Library,
		"model_type":            metadata.ModelType,
		"parameter_count":       metadata.ParameterCount,
		"source_private":        metadata.Private,
		"source_gated":          metadata.Gated,
		"source_login_required": metadata.LoginRequired,
		"source_downloads":      metadata.Downloads,
		"source_likes":          metadata.Likes,
		"metadata_refreshed_at": time.Now(),
	}
	if metadata.SizeBytes > 0 && download.SizeBytes == 0 {
		updates["size_bytes"] = metadata.SizeBytes
	}
	if metadata.UpdatedAt != nil && !metadata.UpdatedAt.IsZero() {
		updates["source_updated_at"] = *metadata.UpdatedAt
	}
	if metadata.CreatedAt != nil && !metadata.CreatedAt.IsZero() {
		updates["source_created_at"] = *metadata.CreatedAt
	}
	if err := db.Model(&model.ModelDownload{}).Where("id = ?", download.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("database update: %w", err)
	}

	var dataset model.Dataset
	if err := db.Where("name = ? AND type = ?", download.Name, model.DataType(download.Category)).First(&dataset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("dataset lookup: %w", err)
	}
	extra := dataset.Extra.Data()
	extra.Tags = mergeTags(extra.Tags, append([]string{string(download.Source)}, metadata.Tags...))
	extra.WebURL = &sourceURL
	datasetSize := download.SizeBytes
	if datasetSize == 0 {
		datasetSize = metadata.SizeBytes
	}
	datasetUpdates := map[string]any{"extra": datatypes.NewJSONType(extra), "size_bytes": datasetSize}
	if metadata.Description != "" && isGeneratedDescription(dataset.Describe, download) {
		datasetUpdates["describe"] = metadata.Description
	}
	if err := db.Model(&model.Dataset{}).Where("id = ?", dataset.ID).
		Updates(datasetUpdates).Error; err != nil {
		return fmt.Errorf("dataset %d metadata update: %w", dataset.ID, err)
	}
	return nil
}

func fetchHuggingFaceAvatar(client *http.Client, owner string) (string, error) {
	escapedOwner := url.PathEscape(owner)
	endpoints := []string{
		"https://huggingface.co/api/organizations/" + escapedOwner + "/overview",
		"https://huggingface.co/api/users/" + escapedOwner + "/overview",
	}
	for _, endpoint := range endpoints {
		response, err := getResponse(client, endpoint)
		if err != nil {
			return "", err
		}
		if response.StatusCode == http.StatusNotFound {
			response.Body.Close()
			continue
		}
		if response.StatusCode != http.StatusOK {
			statusCode := response.StatusCode
			response.Body.Close()
			return "", fmt.Errorf("source returned HTTP %d", statusCode)
		}
		var payload struct {
			AvatarURL string `json:"avatarUrl"`
		}
		decodeErr := json.NewDecoder(response.Body).Decode(&payload)
		response.Body.Close()
		if decodeErr != nil {
			return "", decodeErr
		}
		return payload.AvatarURL, nil
	}
	return "", nil
}

func fetchMetadata(client *http.Client, download *model.ModelDownload) (sourceMetadata, error) {
	owner, name, ok := strings.Cut(download.Name, "/")
	if !ok {
		return sourceMetadata{}, fmt.Errorf("invalid repository name")
	}
	owner = url.PathEscape(owner)
	name = url.PathEscape(name)

	if download.Source == model.ModelSourceHuggingFace {
		resource := "models"
		if download.Category == model.DownloadCategoryDataset {
			resource = "datasets"
		}
		endpoint := fmt.Sprintf("https://huggingface.co/api/%s/%s/%s", resource, owner, name)
		var payload struct {
			Downloads    int64           `json:"downloads"`
			Likes        int64           `json:"likes"`
			LastModified time.Time       `json:"lastModified"`
			CreatedAt    time.Time       `json:"createdAt"`
			Tags         []string        `json:"tags"`
			UsedStorage  int64           `json:"usedStorage"`
			Private      bool            `json:"private"`
			Gated        any             `json:"gated"`
			PipelineTag  string          `json:"pipeline_tag"`
			LibraryName  string          `json:"library_name"`
			CardData     json.RawMessage `json:"cardData"`
			Config       struct {
				ModelType string `json:"model_type"`
			} `json:"config"`
			Safetensors struct {
				Total int64 `json:"total"`
			} `json:"safetensors"`
		}
		if err := getJSON(client, endpoint, &payload); err != nil {
			return sourceMetadata{}, err
		}
		var cardData struct {
			License string `json:"license"`
		}
		_ = json.Unmarshal(payload.CardData, &cardData)
		readmeURL := "https://huggingface.co/"
		if download.Category == model.DownloadCategoryDataset {
			readmeURL += "datasets/"
		}
		readmeURL += owner + "/" + name + "/raw/"
		revision := download.Revision
		if revision == "" {
			revision = "main"
		}
		readmeURL += url.PathEscape(revision) + "/README.md"
		readme, _ := fetchOptionalText(client, readmeURL, maxStoredReadmeBytes)
		readme = cleanReadme(readme)
		return sourceMetadata{
			DisplayName:    name,
			License:        cardData.License,
			Readme:         readme,
			Description:    sourceDescription("", readme),
			Task:           payload.PipelineTag,
			Library:        payload.LibraryName,
			ModelType:      payload.Config.ModelType,
			ParameterCount: payload.Safetensors.Total,
			Private:        payload.Private,
			Gated:          sourceFlag(payload.Gated),
			Downloads:      payload.Downloads,
			Likes:          payload.Likes,
			SizeBytes:      payload.UsedStorage,
			CreatedAt:      &payload.CreatedAt,
			UpdatedAt:      &payload.LastModified,
			Tags:           limitTags(payload.Tags),
		}, nil
	}

	resource := "models"
	if download.Category == model.DownloadCategoryDataset {
		resource = "datasets"
	}
	endpoint := fmt.Sprintf("https://modelscope.cn/openapi/v1/%s/%s/%s", resource, owner, name)
	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			DisplayName   string    `json:"display_name"`
			Description   string    `json:"description"`
			Readme        string    `json:"readme"`
			License       string    `json:"license"`
			Downloads     int64     `json:"downloads"`
			Likes         int64     `json:"likes"`
			FileSize      int64     `json:"file_size"`
			Params        int64     `json:"params"`
			CreatedAt     time.Time `json:"created_at"`
			LastModified  time.Time `json:"last_modified"`
			Tasks         []string  `json:"tasks"`
			Tags          []string  `json:"tags"`
			Private       bool      `json:"private"`
			Gated         bool      `json:"gated"`
			LoginRequired bool      `json:"login_required"`
		} `json:"data"`
	}
	if err := getJSON(client, endpoint, &payload); err != nil {
		return sourceMetadata{}, err
	}
	if !payload.Success {
		return sourceMetadata{}, fmt.Errorf("source returned success=false")
	}
	task := ""
	if len(payload.Data.Tasks) > 0 {
		task = payload.Data.Tasks[0]
	}
	library := tagValue(payload.Data.Tags, "library:")
	modelType := tagValue(payload.Data.Tags, "model_type:")
	return sourceMetadata{
		DisplayName:    payload.Data.DisplayName,
		Description:    sourceDescription(payload.Data.Description, cleanReadme(payload.Data.Readme)),
		Readme:         cleanReadme(payload.Data.Readme),
		License:        payload.Data.License,
		Task:           task,
		Library:        library,
		ModelType:      modelType,
		ParameterCount: payload.Data.Params,
		Private:        payload.Data.Private,
		Gated:          payload.Data.Gated,
		LoginRequired:  payload.Data.LoginRequired,
		Downloads:      payload.Data.Downloads,
		Likes:          payload.Data.Likes,
		SizeBytes:      payload.Data.FileSize,
		CreatedAt:      &payload.Data.CreatedAt,
		UpdatedAt:      &payload.Data.LastModified,
		Tags:           limitTags(append(payload.Data.Tasks, payload.Data.Tags...)),
	}, nil
}

func fetchOptionalText(client *http.Client, endpoint string, limit int) (string, error) {
	response, err := getResponse(client, endpoint)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("source returned HTTP %d", response.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, int64(limit+1)))
	if err != nil {
		return "", err
	}
	return truncateText(string(content), limit), nil
}

func truncateText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	for limit > 0 && !utf8.RuneStart(text[limit]) {
		limit--
	}
	return text[:limit]
}

func cleanReadme(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "---\n") {
		if end := strings.Index(text[4:], "\n---"); end >= 0 {
			text = text[end+8:]
		}
	}
	text = sourceUnsafeHTMLBlockPattern.ReplaceAllString(text, "")
	text = sourceHTMLTagPattern.ReplaceAllString(text, " ")
	text = html.UnescapeString(text)
	return truncateText(strings.TrimSpace(text), maxStoredReadmeBytes)
}

func sourceFlag(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed != "" && typed != "false" && typed != "none"
	default:
		return false
	}
}

func tagValue(tags []string, prefix string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, prefix) {
			return strings.TrimPrefix(tag, prefix)
		}
	}
	return ""
}

func sourceDescription(description, readme string) string {
	text := strings.TrimSpace(description)
	if text == "" {
		text = strings.TrimSpace(readme)
	}
	if text == "" {
		return ""
	}
	text = html.UnescapeString(sourceHTMLTagPattern.ReplaceAllString(text, " "))
	text = sourceMarkdownLinkPattern.ReplaceAllString(text, "$1")
	text = strings.Map(func(character rune) rune {
		if strings.ContainsRune("#*_`|", character) {
			return -1
		}
		return character
	}, text)
	text = strings.Join(strings.Fields(text), " ")
	truncated := truncateText(text, maxSourceDescriptionBytes)
	if len(truncated) < len(text) {
		return truncated + "…"
	}
	return truncated
}

var (
	sourceHTMLTagPattern         = regexp.MustCompile(`<[^>]+>`)
	sourceMarkdownLinkPattern    = regexp.MustCompile(`!?\[([^]]+)]\([^)]+\)`)
	sourceUnsafeHTMLBlockPattern = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
)

func isGeneratedDescription(description string, download *model.ModelDownload) bool {
	description = strings.TrimSpace(description)
	return description == "" || strings.Contains(description, download.Name) &&
		(strings.Contains(description, "ModelScope") || strings.Contains(description, "HuggingFace"))
}

func getJSON(client *http.Client, endpoint string, target any) error {
	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		response, err := getResponse(client, endpoint)
		if err == nil {
			if response.StatusCode == http.StatusOK {
				decodeErr := json.NewDecoder(response.Body).Decode(target)
				response.Body.Close()
				return decodeErr
			}
			lastErr = fmt.Errorf("source returned HTTP %d", response.StatusCode)
			retryable := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError
			response.Body.Close()
			if !retryable {
				return lastErr
			}
		} else {
			lastErr = err
		}
		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	return lastErr
}

func repositoryURL(download *model.ModelDownload) string {
	if download.Source == model.ModelSourceHuggingFace {
		if download.Category == model.DownloadCategoryDataset {
			return "https://huggingface.co/datasets/" + download.Name
		}
		return "https://huggingface.co/" + download.Name
	}
	if download.Category == model.DownloadCategoryDataset {
		return "https://modelscope.cn/datasets/" + download.Name
	}
	return "https://modelscope.cn/models/" + download.Name
}

func limitTags(tags []string) []string {
	result := make([]string, 0, maxMetadataTags)
	for _, tag := range tags {
		if tag == "" || tag == "auto-download" || strings.HasPrefix(tag, "license:") {
			continue
		}
		result = append(result, tag)
		if len(result) == maxMetadataTags {
			break
		}
	}
	return result
}

func getResponse(client *http.Client, endpoint string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	return client.Do(request)
}

func mergeTags(existing, discovered []string) []string {
	result := make([]string, 0, len(existing)+len(discovered))
	seen := make(map[string]struct{}, len(existing)+len(discovered))
	for _, tag := range append(existing, discovered...) {
		if tag == "" || tag == "auto-download" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
	}
	return result
}
