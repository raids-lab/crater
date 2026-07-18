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
	stdhtml "html"
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
	"github.com/raids-lab/crater/internal/governance/modeldataset"
	"github.com/raids-lab/crater/pkg/config"
)

const (
	defaultBatchSize          = 100
	maxBatchSize              = 1000
	defaultRequestDelay       = 100 * time.Millisecond
	maxSourceDescriptionBytes = 500
	maxMetadataTags           = 4
	maxStoredReadmeBytes      = modeldataset.MaxStoredReadmeBytes
	maxModelScopePageBytes    = 2 * 1024 * 1024
	maxLogoRedirects          = 5
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

type sourceEndpoints struct {
	HuggingFace []string
	ModelScope  []string
}

type cachedLogo struct {
	URL         string
	Data        []byte
	ContentType string
}

//nolint:gocyclo // Flag-driven batch orchestration is intentionally kept linear for resumability.
func main() {
	apply := flag.Bool("apply", false, "write refreshed metadata to the database")
	batchSize := flag.Int("batch-size", defaultBatchSize, "number of records loaded from the database per batch")
	afterID := flag.Uint("after-id", 0, "resume after this model download ID")
	maxRecords := flag.Int("max-records", 0, "maximum records to process; 0 means unlimited")
	force := flag.Bool("force", false, "refresh records even if their metadata is still fresh")
	missingLogoOnly := flag.Bool("missing-logo-only", false, "refresh only records without a cached source logo")
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

	appConfig := config.GetConfig()
	db := query.GetDB()
	client := &http.Client{Timeout: time.Duration(appConfig.MetadataTimeoutSeconds()) * time.Second}
	endpoints := sourceEndpoints{
		HuggingFace: appConfig.HuggingFaceMetadataEndpoints(),
		ModelScope:  appConfig.ModelScopeMetadataEndpoints(),
	}
	avatarCache := make(map[string]cachedLogo)
	var processed, refreshed, failed int
	cursor := *afterID
	stop := false
	for !stop {
		var downloads []model.ModelDownload
		builder := db.Where("status = ? AND id > ?", model.ModelDownloadStatusReady, cursor)
		if *source != "" {
			builder = builder.Where("source = ?", *source)
		}
		if *missingLogoOnly {
			builder = builder.Where(`
				model_dataset_source_id IS NULL OR EXISTS (
					SELECT 1 FROM model_dataset_sources
					WHERE model_dataset_sources.id = model_downloads.model_dataset_source_id
						AND model_dataset_sources.deleted_at IS NULL
						AND COALESCE(octet_length(model_dataset_sources.logo_data), 0) = 0
				)`)
		} else if !*force {
			builder = builder.Where(`
				metadata_refreshed_at IS NULL OR metadata_refreshed_at < ? OR
				EXISTS (
					SELECT 1 FROM model_dataset_sources
					WHERE model_dataset_sources.id = model_downloads.model_dataset_source_id
						AND model_dataset_sources.deleted_at IS NULL
						AND COALESCE(octet_length(model_dataset_sources.logo_data), 0) = 0
				)`, time.Now().Add(-*staleAfter))
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
			if err := refreshDownload(
				db, client, endpoints, avatarCache, appConfig.MetadataLogoAllowedHosts(),
				appConfig.MetadataMaxLogoBytes(), download, *apply,
			); err != nil {
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
	endpoints sourceEndpoints,
	avatarCache map[string]cachedLogo,
	logoAllowedHosts []string,
	maxLogoBytes int64,
	download *model.ModelDownload,
	apply bool,
) error {
	metadata, selectedEndpoint, err := fetchMetadata(client, endpoints, download)
	if err != nil {
		return err
	}

	organization := strings.SplitN(download.Name, "/", 2)[0]
	organizationKey := strings.ToLower(organization)
	logo, ok := avatarCache[organizationKey]
	if !ok {
		logo, ok = loadCachedOrganizationLogo(db, organization)
		if !ok {
			switch download.Source {
			case model.ModelSourceHuggingFace:
				logo.URL, err = fetchHuggingFaceAvatar(client, endpoints.HuggingFace, organization)
			case model.ModelSourceModelScope:
				logo.URL, err = fetchModelScopeAvatar(client, selectedEndpoint, download)
			}
			if err != nil {
				fmt.Printf("WARN id=%d owner avatar lookup failed: %v\n", download.ID, err)
			}
			if logo.URL != "" {
				logo.Data, logo.ContentType, err = fetchLogo(
					client, logo.URL, logoAllowedHosts, maxLogoBytes,
				)
				if err != nil {
					fmt.Printf("WARN id=%d owner avatar cache failed: %v\n", download.ID, err)
					// Do not persist an untrusted URL or overwrite a previously cached logo
					// when a transient lookup, redirect, or allowlist check fails.
					logo = cachedLogo{}
				}
			}
		}
		avatarCache[organizationKey] = logo
	}
	sourceURL := repositoryURL(download, selectedEndpoint)
	fmt.Printf("OK   id=%d source=%s name=%s downloads=%d likes=%d tags=%v\n",
		download.ID, download.Source, download.Name, metadata.Downloads, metadata.Likes, metadata.Tags)
	if !apply {
		return nil
	}

	updates := map[string]any{
		"organization":          organization,
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
	if logo.URL != "" {
		updates["logo_url"] = logo.URL
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
	return db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		sourceRecord := model.ModelDatasetSource{
			Provider:            model.ModelDatasetProvider(download.Source),
			ResourceType:        model.DataType(download.Category),
			RepositoryID:        download.Name,
			RepositoryURL:       sourceURL,
			Organization:        organization,
			LogoURL:             logo.URL,
			LogoData:            logo.Data,
			LogoContentType:     logo.ContentType,
			DisplayName:         metadata.DisplayName,
			Description:         metadata.Description,
			Readme:              metadata.Readme,
			License:             metadata.License,
			Task:                metadata.Task,
			Library:             metadata.Library,
			ModelType:           metadata.ModelType,
			ParameterCount:      metadata.ParameterCount,
			Private:             metadata.Private,
			Gated:               metadata.Gated,
			LoginRequired:       metadata.LoginRequired,
			Downloads:           metadata.Downloads,
			Likes:               metadata.Likes,
			SourceCreatedAt:     metadata.CreatedAt,
			SourceUpdatedAt:     metadata.UpdatedAt,
			MetadataRefreshedAt: &now,
		}
		var persisted model.ModelDatasetSource
		lookup := tx.Where(
			"provider = ? AND resource_type = ? AND repository_id = ?",
			sourceRecord.Provider, sourceRecord.ResourceType, sourceRecord.RepositoryID,
		).First(&persisted)
		if errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
			if err := tx.Create(&sourceRecord).Error; err != nil {
				return fmt.Errorf("create source record: %w", err)
			}
			persisted = sourceRecord
		} else if lookup.Error != nil {
			return fmt.Errorf("load source record: %w", lookup.Error)
		} else if err := tx.Model(&persisted).Updates(sourceRecord).Error; err != nil {
			return fmt.Errorf("update source record: %w", err)
		}

		updates["model_dataset_source_id"] = persisted.ID
		if err := tx.Model(&model.ModelDownload{}).Where("id = ?", download.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("database update: %w", err)
		}

		physicalPath, public := modeldataset.PhysicalStoragePath(
			download.Path,
			config.GetConfig().MetadataLogicalPublicPrefix(),
			config.GetConfig().Storage.Prefix.Public,
		)
		dataset, err := findDatasetForDownload(tx, download, physicalPath, public)
		if err != nil {
			return err
		}
		if dataset == nil {
			return nil
		}
		extra := dataset.Extra.Data()
		extra.Tags = mergeTags(extra.Tags, append([]string{string(download.Source)}, metadata.Tags...))
		extra.WebURL = &sourceURL
		datasetSize := download.SizeBytes
		if datasetSize == 0 {
			datasetSize = metadata.SizeBytes
		}
		datasetUpdates := map[string]any{
			"extra": datatypes.NewJSONType(extra), "size_bytes": datasetSize,
			"model_dataset_source_id": persisted.ID,
		}
		if metadata.Description != "" && isGeneratedDescription(dataset.Describe, download) {
			datasetUpdates["describe"] = metadata.Description
		}
		if err := tx.Model(&model.Dataset{}).Where("id = ?", dataset.ID).Updates(datasetUpdates).Error; err != nil {
			return fmt.Errorf("dataset %d metadata update: %w", dataset.ID, err)
		}
		return nil
	})
}

func findDatasetForDownload(
	db *gorm.DB,
	download *model.ModelDownload,
	physicalPath string,
	public bool,
) (*model.Dataset, error) {
	if public {
		var exact model.Dataset
		err := db.Where("url = ? AND type = ?", physicalPath, model.DataType(download.Category)).First(&exact).Error
		if err == nil {
			return &exact, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("dataset path lookup: %w", err)
		}
	}

	var matches []model.Dataset
	if err := db.Where(
		"name = ? AND type = ? AND model_dataset_source_id IS NULL",
		download.Name, model.DataType(download.Category),
	).Order("id ASC").Limit(2).Find(&matches).Error; err != nil {
		return nil, fmt.Errorf("dataset identity lookup: %w", err)
	}
	if len(matches) != 1 {
		return nil, nil
	}
	return &matches[0], nil
}

func fetchHuggingFaceAvatar(client *http.Client, baseEndpoints []string, owner string) (string, error) {
	return modeldataset.FetchHuggingFaceAvatarURL(context.Background(), client, baseEndpoints, owner)
}

func fetchModelScopeAvatar(
	client *http.Client, baseEndpoint string, download *model.ModelDownload,
) (string, error) {
	return modeldataset.FetchModelScopeAvatarURL(
		context.Background(), client, repositoryURL(download, baseEndpoint),
	)
}

func loadCachedOrganizationLogo(db *gorm.DB, organization string) (cachedLogo, bool) {
	var sources []model.ModelDatasetSource
	err := db.Where(
		"LOWER(organization) = ? AND octet_length(logo_data) > 0", strings.ToLower(organization),
	).Order("updated_at DESC").Limit(1).Find(&sources).Error
	if err != nil || len(sources) == 0 {
		return cachedLogo{}, false
	}
	source := sources[0]
	return cachedLogo{URL: source.LogoURL, Data: source.LogoData, ContentType: source.LogoContentType}, true
}

func fetchMetadata(
	client *http.Client, endpoints sourceEndpoints, download *model.ModelDownload,
) (sourceMetadata, string, error) {
	owner, name, ok := strings.Cut(download.Name, "/")
	if !ok {
		return sourceMetadata{}, "", fmt.Errorf("invalid repository name")
	}
	owner = url.PathEscape(owner)
	name = url.PathEscape(name)

	if download.Source == model.ModelSourceHuggingFace {
		resource := "models"
		if download.Category == model.DownloadCategoryDataset {
			resource = "datasets"
		}
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
		selectedEndpoint, err := getJSONFromEndpoints(
			client, endpoints.HuggingFace, "/api/"+resource+"/"+owner+"/"+name, &payload,
		)
		if err != nil {
			return sourceMetadata{}, "", err
		}
		var cardData struct {
			License string `json:"license"`
		}
		_ = json.Unmarshal(payload.CardData, &cardData)
		readmeURL := selectedEndpoint + "/"
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
		}, selectedEndpoint, nil
	}

	resource := "models"
	if download.Category == model.DownloadCategoryDataset {
		resource = "datasets"
	}
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
	selectedEndpoint, err := getJSONFromEndpoints(
		client, endpoints.ModelScope, "/openapi/v1/"+resource+"/"+owner+"/"+name, &payload,
	)
	if err != nil {
		return sourceMetadata{}, "", err
	}
	if !payload.Success {
		return sourceMetadata{}, "", fmt.Errorf("source returned success=false")
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
	}, selectedEndpoint, nil
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
	return modeldataset.CleanReadme(text, maxStoredReadmeBytes)
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
	text = stdhtml.UnescapeString(sourceHTMLTagPattern.ReplaceAllString(text, " "))
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
	sourceHTMLTagPattern      = regexp.MustCompile(`<[^>]+>`)
	sourceMarkdownLinkPattern = regexp.MustCompile(`!?\[([^]]+)]\([^)]+\)`)
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

func getJSONFromEndpoints(
	client *http.Client, baseEndpoints []string, path string, target any,
) (string, error) {
	var lastErr error
	for _, baseEndpoint := range baseEndpoints {
		baseEndpoint = strings.TrimRight(baseEndpoint, "/")
		if err := getJSON(client, baseEndpoint+path, target); err != nil {
			lastErr = err
			continue
		}
		return baseEndpoint, nil
	}
	if lastErr == nil {
		lastErr = errors.New("no source metadata endpoint configured")
	}
	return "", lastErr
}

func fetchLogo(client *http.Client, endpoint string, allowedHosts []string, maxBytes int64) (
	data []byte, contentType string, err error,
) {
	return modeldataset.FetchSourceLogo(
		context.Background(), client, endpoint, allowedHosts, maxBytes,
	)
}

func repositoryURL(download *model.ModelDownload, baseEndpoint string) string {
	baseEndpoint = strings.TrimRight(baseEndpoint, "/")
	if download.Source == model.ModelSourceHuggingFace {
		if download.Category == model.DownloadCategoryDataset {
			return baseEndpoint + "/datasets/" + download.Name
		}
		return baseEndpoint + "/" + download.Name
	}
	if download.Category == model.DownloadCategoryDataset {
		return baseEndpoint + "/datasets/" + download.Name
	}
	return baseEndpoint + "/models/" + download.Name
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
