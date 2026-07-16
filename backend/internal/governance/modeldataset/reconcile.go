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

package modeldataset

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

type ReconcileOptions struct {
	Apply                 bool
	LogicalPublicPrefix   string
	PhysicalPublicPrefix  string
	PhysicalUserPrefix    string
	PhysicalAccountPrefix string
	MaxReadmeBytes        int
	Now                   time.Time
}

type ReconcileReport struct {
	ReadyDownloads           int
	PublicDownloads          int
	SourceIdentities         int
	DownloadLinks            int
	DatasetLinks             int
	Candidates               int
	RegisteredCandidates     int
	UnregisteredCandidates   int
	ReadmesFromStorage       int
	DiscoveriesWritten       int
	RegisteredNonPublic      int
	PathlessDatasets         int
	MissingMarked            int64
	ProvenanceHighConfidence int
	ProvenanceHints          int
	InferredSources          int
	OwnerMatches             int
}

type sourceIdentity struct {
	Provider     model.ModelDatasetProvider
	ResourceType model.DataType
	RepositoryID string
}

//nolint:gocyclo,funlen // Reconciliation reports and applies one idempotent plan in a single pass.
func ReconcilePublic(
	ctx context.Context,
	db *gorm.DB,
	candidates []Candidate,
	options *ReconcileOptions,
) (ReconcileReport, error) {
	if options.LogicalPublicPrefix == "" || options.PhysicalPublicPrefix == "" {
		return ReconcileReport{}, errors.New("logical and physical public prefixes are required")
	}
	if options.MaxReadmeBytes < 1 {
		return ReconcileReport{}, errors.New("max README bytes must be positive")
	}
	if options.Now.IsZero() {
		options.Now = time.Now()
	}

	var downloads []*model.ModelDownload
	if err := db.WithContext(ctx).
		Where("status = ?", model.ModelDownloadStatusReady).
		Order("id ASC").
		Find(&downloads).Error; err != nil {
		return ReconcileReport{}, fmt.Errorf("load ready downloads: %w", err)
	}
	var datasets []*model.Dataset
	if err := db.WithContext(ctx).
		Where("deleted_at IS NULL AND type IN ?", []model.DataType{model.DataTypeModel, model.DataTypeDataset}).
		Find(&datasets).Error; err != nil {
		return ReconcileReport{}, fmt.Errorf("load datasets: %w", err)
	}

	report := ReconcileReport{ReadyDownloads: len(downloads), Candidates: len(candidates)}
	datasetsByPath := make(map[string][]*model.Dataset)
	datasetsByResource := make(map[string][]*model.Dataset)
	for _, dataset := range datasets {
		datasetsByPath[cleanStoragePath(dataset.URL)] = append(datasetsByPath[cleanStoragePath(dataset.URL)], dataset)
		key := resourceKey(dataset.Name, dataset.Type)
		datasetsByResource[key] = append(datasetsByResource[key], dataset)
	}
	ownerMatches, err := enrichCandidateOwners(ctx, db, candidates)
	if err != nil {
		return report, err
	}
	report.OwnerMatches = ownerMatches
	candidatesByPath := make(map[string]*Candidate, len(candidates))
	for index := range candidates {
		candidate := &candidates[index]
		candidatesByPath[cleanStoragePath(candidate.Path)] = candidate
	}

	seenIdentities := make(map[sourceIdentity]struct{})
	sourceByPath := make(map[string]*model.ModelDatasetSource)
	sourceIDs := make([]uint, 0)
	for _, dataset := range datasets {
		if dataset.ModelDatasetSourceID != nil {
			sourceIDs = append(sourceIDs, *dataset.ModelDatasetSourceID)
		}
	}
	if len(sourceIDs) > 0 {
		var persistedSources []*model.ModelDatasetSource
		if err := db.WithContext(ctx).Where("id IN ?", sourceIDs).Find(&persistedSources).Error; err != nil {
			return report, fmt.Errorf("load linked sources: %w", err)
		}
		sourcesByID := make(map[uint]*model.ModelDatasetSource, len(persistedSources))
		for _, source := range persistedSources {
			sourcesByID[source.ID] = source
		}
		for _, dataset := range datasets {
			if dataset.ModelDatasetSourceID == nil {
				continue
			}
			if source := sourcesByID[*dataset.ModelDatasetSourceID]; source != nil {
				sourceByPath[cleanStoragePath(dataset.URL)] = source
				seenIdentities[sourceIdentity{
					Provider: source.Provider, ResourceType: source.ResourceType, RepositoryID: source.RepositoryID,
				}] = struct{}{}
			}
		}
	}
	for _, download := range downloads {
		physicalPath, public := PhysicalStoragePath(
			download.Path,
			options.LogicalPublicPrefix,
			options.PhysicalPublicPrefix,
		)
		if !public {
			continue
		}
		report.PublicDownloads++
		identity := sourceIdentity{
			Provider:     model.ModelDatasetProvider(download.Source),
			ResourceType: model.DataType(download.Category),
			RepositoryID: download.Name,
		}
		seenIdentities[identity] = struct{}{}

		source := sourceFromDownload(download)
		if candidate, ok := candidatesByPath[physicalPath]; ok {
			readme, err := readLocalReadme(candidate.AbsolutePath, options.MaxReadmeBytes)
			if err != nil {
				return report, fmt.Errorf("read README for %s: %w", physicalPath, err)
			}
			if readme != "" {
				source.Readme = readme
				report.ReadmesFromStorage++
			}
		}

		if options.Apply {
			persisted, err := upsertSource(ctx, db, source)
			if err != nil {
				return report, err
			}
			source = persisted
			if err := db.WithContext(ctx).Model(&model.ModelDownload{}).
				Where("id = ?", download.ID).
				Update("model_dataset_source_id", source.ID).Error; err != nil {
				return report, fmt.Errorf("link download %d to source: %w", download.ID, err)
			}
			report.DownloadLinks++
			linkedDataset := false
			for _, dataset := range datasetsByPath[physicalPath] {
				if dataset.Type != identity.ResourceType {
					continue
				}
				if err := db.WithContext(ctx).Model(&model.Dataset{}).
					Where("id = ?", dataset.ID).
					Update("model_dataset_source_id", source.ID).Error; err != nil {
					return report, fmt.Errorf("link dataset %d to source: %w", dataset.ID, err)
				}
				dataset.ModelDatasetSourceID = &source.ID
				report.DatasetLinks++
				linkedDataset = true
			}
			// Historical shared records may have an empty or non-standard path. Link only
			// when repository identity and type select one unambiguous Dataset row.
			if !linkedDataset {
				matches := datasetsByResource[resourceKey(download.Name, identity.ResourceType)]
				if len(matches) == 1 && matches[0].ModelDatasetSourceID == nil {
					dataset := matches[0]
					if err := db.WithContext(ctx).Model(&model.Dataset{}).
						Where("id = ?", dataset.ID).
						Update("model_dataset_source_id", source.ID).Error; err != nil {
						return report, fmt.Errorf("link pathless dataset %d to source: %w", dataset.ID, err)
					}
					dataset.ModelDatasetSourceID = &source.ID
					report.DatasetLinks++
				}
			}
		}
		sourceByPath[physicalPath] = source
	}
	seenPaths := make([]string, 0, len(candidates))
	for index := range candidates {
		candidate := &candidates[index]
		path := cleanStoragePath(candidate.Path)
		seenPaths = append(seenPaths, path)
		if candidate.Evidence.ProvenanceConfidence == provenanceConfidenceHigh {
			report.ProvenanceHighConfidence++
		} else if candidate.Evidence.RepositoryID != "" || candidate.Evidence.ConfigNameOrPath != "" ||
			len(candidate.Evidence.CandidateURLs) > 0 {
			report.ProvenanceHints++
		}
		status := model.ModelDatasetDiscoveryStatusDiscovered
		var datasetID, sourceID *uint
		if matches := datasetsByPath[path]; len(matches) > 0 {
			status = model.ModelDatasetDiscoveryStatusRegistered
			datasetID = &matches[0].ID
			report.RegisteredCandidates++
		} else {
			report.UnregisteredCandidates++
		}
		source := sourceByPath[path]
		if source == nil && canInferSource(&candidate.Evidence) {
			report.InferredSources++
			identity := sourceIdentity{
				Provider: candidate.Evidence.Provider, ResourceType: candidate.Type,
				RepositoryID: candidate.Evidence.RepositoryID,
			}
			seenIdentities[identity] = struct{}{}
			inferred := sourceFromCandidate(candidate)
			if options.Apply {
				readme, err := readLocalReadme(candidate.AbsolutePath, options.MaxReadmeBytes)
				if err != nil {
					return report, fmt.Errorf("read README for %s: %w", path, err)
				}
				if readme != "" {
					inferred.Readme = readme
					report.ReadmesFromStorage++
				}
				persisted, err := upsertSource(ctx, db, inferred)
				if err != nil {
					return report, err
				}
				inferred = persisted
				for _, dataset := range datasetsByPath[path] {
					if dataset.Type != candidate.Type || dataset.ModelDatasetSourceID != nil {
						continue
					}
					if err := db.WithContext(ctx).Model(&model.Dataset{}).Where("id = ?", dataset.ID).
						Update("model_dataset_source_id", inferred.ID).Error; err != nil {
						return report, fmt.Errorf("link dataset %d to inferred source: %w", dataset.ID, err)
					}
					dataset.ModelDatasetSourceID = &inferred.ID
					report.DatasetLinks++
				}
			}
			source = inferred
			sourceByPath[path] = inferred
		}
		if source != nil && source.ID != 0 {
			sourceID = &source.ID
		}
		if !options.Apply {
			continue
		}
		discovery := model.ModelDatasetDiscovery{
			DiscoveryKey: "path:" + path,
			Path:         path,
			Scope:        model.ModelDatasetDiscoveryScopePublic,
			DetectedType: candidate.Type,
			DetectedName: candidate.Name,
			Evidence:     datatypes.NewJSONType(candidate.Evidence),
			SizeBytes:    candidate.SizeBytes,
			DatasetID:    datasetID,
			SourceID:     sourceID,
			Status:       status,
			FirstSeenAt:  options.Now,
			LastSeenAt:   options.Now,
		}
		if err := upsertDiscovery(ctx, db, &discovery); err != nil {
			return report, err
		}
		report.DiscoveriesWritten++
	}

	for _, dataset := range datasets {
		path := cleanStoragePath(dataset.URL)
		if path == "" {
			report.PathlessDatasets++
			if options.Apply {
				datasetID := dataset.ID
				ownerID := dataset.UserID
				discovery := model.ModelDatasetDiscovery{
					DiscoveryKey: "dataset:" + strconv.FormatUint(uint64(dataset.ID), 10),
					Scope:        model.ModelDatasetDiscoveryScopeUser, ScopeID: &ownerID,
					DetectedType: dataset.Type, DetectedName: dataset.Name,
					DatasetID: &datasetID, SourceID: dataset.ModelDatasetSourceID,
					Status:      model.ModelDatasetDiscoveryStatusPathMissing,
					FirstSeenAt: options.Now, LastSeenAt: options.Now,
				}
				if err := upsertDiscovery(ctx, db, &discovery); err != nil {
					return report, err
				}
				report.DiscoveriesWritten++
			}
			continue
		}
		if pathUnderPrefix(path, options.PhysicalPublicPrefix) {
			continue
		}
		scope := model.ModelDatasetDiscoveryScopeUser
		var scopeID *uint
		if pathUnderPrefix(path, options.PhysicalAccountPrefix) {
			scope = model.ModelDatasetDiscoveryScopeAccount
		} else {
			ownerID := dataset.UserID
			scopeID = &ownerID
		}
		report.RegisteredNonPublic++
		if !options.Apply {
			continue
		}
		datasetID := dataset.ID
		discovery := model.ModelDatasetDiscovery{
			DiscoveryKey: "dataset:" + strconv.FormatUint(uint64(dataset.ID), 10),
			Path:         path, Scope: scope, ScopeID: scopeID,
			DetectedType: dataset.Type, DetectedName: dataset.Name,
			DatasetID: &datasetID, SourceID: dataset.ModelDatasetSourceID,
			Status:      model.ModelDatasetDiscoveryStatusRegistered,
			FirstSeenAt: options.Now, LastSeenAt: options.Now,
		}
		if err := upsertDiscovery(ctx, db, &discovery); err != nil {
			return report, err
		}
		report.DiscoveriesWritten++
	}

	if options.Apply {
		query := db.WithContext(ctx).Model(&model.ModelDatasetDiscovery{}).
			Where("scope = ?", model.ModelDatasetDiscoveryScopePublic).
			Where("path = ? OR path LIKE ?", options.PhysicalPublicPrefix, options.PhysicalPublicPrefix+"/%")
		if len(seenPaths) > 0 {
			query = query.Where("path NOT IN ?", seenPaths)
		}
		result := query.Updates(map[string]any{
			"status":       model.ModelDatasetDiscoveryStatusMissing,
			"last_seen_at": options.Now,
		})
		if result.Error != nil {
			return report, fmt.Errorf("mark missing discoveries: %w", result.Error)
		}
		report.MissingMarked = result.RowsAffected
	}
	report.SourceIdentities = len(seenIdentities)
	return report, nil
}

func canInferSource(evidence *model.ModelDatasetDiscoveryEvidence) bool {
	return evidence.ProvenanceConfidence == provenanceConfidenceHigh && evidence.RepositoryID != "" &&
		(evidence.Provider == model.ModelDatasetProviderHuggingFace ||
			evidence.Provider == model.ModelDatasetProviderModelScope)
}

func sourceFromCandidate(candidate *Candidate) *model.ModelDatasetSource {
	organization := ""
	if parts := strings.Split(candidate.Evidence.RepositoryID, "/"); len(parts) == 2 {
		organization = parts[0]
	}
	return &model.ModelDatasetSource{
		Provider: candidate.Evidence.Provider, ResourceType: candidate.Type,
		RepositoryID: candidate.Evidence.RepositoryID, RepositoryURL: candidate.Evidence.RepositoryURL,
		Organization: organization, DisplayName: candidate.Name,
	}
}

func enrichCandidateOwners(ctx context.Context, db *gorm.DB, candidates []Candidate) (int, error) {
	if !db.Migrator().HasTable(&model.User{}) {
		return 0, nil
	}
	var users []*model.User
	if err := db.WithContext(ctx).Select("id", "name", "attributes").Find(&users).Error; err != nil {
		return 0, fmt.Errorf("load filesystem user identities: %w", err)
	}
	usersByUID := make(map[string][]*model.User)
	for _, user := range users {
		attributes := user.Attributes.Data()
		if attributes.UID == nil {
			continue
		}
		uid := strings.TrimSpace(*attributes.UID)
		if uid != "" {
			usersByUID[uid] = append(usersByUID[uid], user)
		}
	}
	matches := 0
	for index := range candidates {
		candidate := &candidates[index]
		users := usersByUID[candidate.Evidence.FilesystemUID]
		if len(users) != 1 {
			continue
		}
		ownerID := users[0].ID
		candidate.Evidence.OwnerUserID = &ownerID
		candidate.Evidence.OwnerUsername = users[0].Name
		matches++
	}
	return matches, nil
}

func resourceKey(name string, resourceType model.DataType) string {
	return string(resourceType) + "\x00" + name
}

func pathUnderPrefix(path, prefix string) bool {
	prefix = cleanStoragePath(prefix)
	if prefix == "" {
		return false
	}
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func PhysicalStoragePath(path, logicalPublicPrefix, physicalPublicPrefix string) (string, bool) {
	path = cleanStoragePath(path)
	logicalPublicPrefix = cleanStoragePath(logicalPublicPrefix)
	physicalPublicPrefix = cleanStoragePath(physicalPublicPrefix)
	if path == logicalPublicPrefix {
		return physicalPublicPrefix, true
	}
	if strings.HasPrefix(path, logicalPublicPrefix+"/") {
		return physicalPublicPrefix + strings.TrimPrefix(path, logicalPublicPrefix), true
	}
	if path == physicalPublicPrefix || strings.HasPrefix(path, physicalPublicPrefix+"/") {
		return path, true
	}
	return path, false
}

// LogicalStoragePath converts a physical storage prefix back to the stable logical
// prefix used by the file-system API and user-facing mount paths.
func LogicalStoragePath(path, logicalPrefix, physicalPrefix string) (string, bool) {
	path = cleanStoragePath(path)
	logicalPrefix = cleanStoragePath(logicalPrefix)
	physicalPrefix = cleanStoragePath(physicalPrefix)
	if path == logicalPrefix || strings.HasPrefix(path, logicalPrefix+"/") {
		return path, true
	}
	if path == physicalPrefix {
		return logicalPrefix, true
	}
	if strings.HasPrefix(path, physicalPrefix+"/") {
		return logicalPrefix + strings.TrimPrefix(path, physicalPrefix), true
	}
	return path, false
}

func sourceFromDownload(download *model.ModelDownload) *model.ModelDatasetSource {
	return &model.ModelDatasetSource{
		Provider:            model.ModelDatasetProvider(download.Source),
		ResourceType:        model.DataType(download.Category),
		RepositoryID:        download.Name,
		RepositoryURL:       download.SourceURL,
		Organization:        download.Organization,
		LogoURL:             download.LogoURL,
		DisplayName:         download.DisplayName,
		Description:         download.SourceDescription,
		Readme:              download.SourceReadme,
		License:             download.License,
		Task:                download.Task,
		Library:             download.Library,
		ModelType:           download.ModelType,
		ParameterCount:      download.ParameterCount,
		Private:             download.SourcePrivate,
		Gated:               download.SourceGated,
		LoginRequired:       download.SourceLoginRequired,
		Downloads:           download.SourceDownloads,
		Likes:               download.SourceLikes,
		SourceCreatedAt:     download.SourceCreatedAt,
		SourceUpdatedAt:     download.SourceUpdatedAt,
		MetadataRefreshedAt: download.MetadataRefreshedAt,
	}
}

func upsertSource(
	ctx context.Context,
	db *gorm.DB,
	source *model.ModelDatasetSource,
) (*model.ModelDatasetSource, error) {
	var existing model.ModelDatasetSource
	err := db.WithContext(ctx).Where(
		"provider = ? AND resource_type = ? AND repository_id = ?",
		source.Provider, source.ResourceType, source.RepositoryID,
	).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.WithContext(ctx).Create(source).Error; err != nil {
			return nil, fmt.Errorf("create source %s/%s/%s: %w", source.Provider, source.ResourceType, source.RepositoryID, err)
		}
		return source, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load source %s/%s/%s: %w", source.Provider, source.ResourceType, source.RepositoryID, err)
	}
	updates := nonEmptySourceUpdates(source)
	if len(updates) > 0 {
		if err := db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("update source %d: %w", existing.ID, err)
		}
	}
	return &existing, nil
}

func nonEmptySourceUpdates(source *model.ModelDatasetSource) map[string]any {
	updates := make(map[string]any)
	stringsByColumn := map[string]string{
		"repository_url": source.RepositoryURL,
		"organization":   source.Organization,
		"logo_url":       source.LogoURL,
		"display_name":   source.DisplayName,
		"description":    source.Description,
		"readme":         source.Readme,
		"license":        source.License,
		"task":           source.Task,
		"library":        source.Library,
		"model_type":     source.ModelType,
	}
	for column, value := range stringsByColumn {
		if value != "" {
			updates[column] = value
		}
	}
	if source.ParameterCount > 0 {
		updates["parameter_count"] = source.ParameterCount
	}
	if source.Downloads > 0 {
		updates["downloads"] = source.Downloads
	}
	if source.Likes > 0 {
		updates["likes"] = source.Likes
	}
	if source.Private {
		updates["private"] = true
	}
	if source.Gated {
		updates["gated"] = true
	}
	if source.LoginRequired {
		updates["login_required"] = true
	}
	if source.SourceCreatedAt != nil {
		updates["source_created_at"] = source.SourceCreatedAt
	}
	if source.SourceUpdatedAt != nil {
		updates["source_updated_at"] = source.SourceUpdatedAt
	}
	if source.MetadataRefreshedAt != nil {
		updates["metadata_refreshed_at"] = source.MetadataRefreshedAt
	}
	return updates
}

func upsertDiscovery(ctx context.Context, db *gorm.DB, discovery *model.ModelDatasetDiscovery) error {
	var existing model.ModelDatasetDiscovery
	err := db.WithContext(ctx).Where("discovery_key = ?", discovery.DiscoveryKey).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.WithContext(ctx).Create(discovery).Error; err != nil {
			return fmt.Errorf("create discovery %s: %w", discovery.Path, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("load discovery %s: %w", discovery.Path, err)
	}
	if err := db.WithContext(ctx).Model(&existing).Updates(map[string]any{
		"path":          discovery.Path,
		"scope":         discovery.Scope,
		"scope_id":      discovery.ScopeID,
		"detected_type": discovery.DetectedType,
		"detected_name": discovery.DetectedName,
		"evidence":      discovery.Evidence,
		"size_bytes":    discovery.SizeBytes,
		"dataset_id":    discovery.DatasetID,
		"source_id":     discovery.SourceID,
		"status":        discovery.Status,
		"last_seen_at":  discovery.LastSeenAt,
	}).Error; err != nil {
		return fmt.Errorf("update discovery %s: %w", discovery.Path, err)
	}
	return nil
}

func readLocalReadme(directory string, maxBytes int) (string, error) {
	for _, name := range []string{"README.md", "readme.md", "README.MD", "README"} {
		path := filepath.Join(directory, name)
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if len(data) > maxBytes {
			data = data[:maxBytes]
			for len(data) > 0 && !utf8.Valid(data) {
				data = data[:len(data)-1]
			}
		}
		return string(data), nil
	}
	return "", nil
}

func cleanStoragePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))), "/")
	if path == "." {
		return ""
	}
	return path
}
