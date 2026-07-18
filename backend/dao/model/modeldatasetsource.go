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

package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ModelDatasetProvider string

const (
	ModelDatasetProviderHuggingFace ModelDatasetProvider = "huggingface"
	ModelDatasetProviderModelScope  ModelDatasetProvider = "modelscope"
	ModelDatasetProviderExternal    ModelDatasetProvider = "external"
)

// ModelDatasetSource stores upstream repository identity and cached metadata.
// Storage location and sharing remain responsibilities of Dataset.
type ModelDatasetSource struct {
	gorm.Model
	Provider      ModelDatasetProvider `gorm:"type:varchar(64);not null;uniqueIndex:idx_model_dataset_source_identity,priority:1"`
	ResourceType  DataType             `gorm:"type:varchar(32);not null;uniqueIndex:idx_model_dataset_source_identity,priority:2"`
	RepositoryID  string               `gorm:"type:varchar(256);not null;uniqueIndex:idx_model_dataset_source_identity,priority:3"`
	RepositoryURL string               `gorm:"type:varchar(512);comment:外部仓库页面地址"`

	Organization        string     `gorm:"type:varchar(128);comment:源站组织或作者"`
	LogoURL             string     `gorm:"type:varchar(512);comment:源站组织头像地址"`
	LogoData            []byte     `gorm:"type:bytea;comment:平台缓存的源站头像"`
	LogoContentType     string     `gorm:"type:varchar(128);comment:平台缓存头像的Content-Type"`
	DisplayName         string     `gorm:"type:varchar(256);comment:源站展示名称"`
	Description         string     `gorm:"type:text;comment:源站简介摘要"`
	Readme              string     `gorm:"type:text;comment:源站README内容(截断保存)"`
	License             string     `gorm:"type:varchar(128);comment:源站许可证"`
	Task                string     `gorm:"type:varchar(128);comment:源站任务分类"`
	Library             string     `gorm:"type:varchar(128);comment:源站框架或库"`
	ModelType           string     `gorm:"type:varchar(128);comment:源站模型类型"`
	ParameterCount      int64      `gorm:"default:0;comment:模型参数量"`
	Private             bool       `gorm:"default:false;comment:源站是否私有"`
	Gated               bool       `gorm:"default:false;comment:源站是否需要申请访问"`
	LoginRequired       bool       `gorm:"default:false;comment:源站是否要求登录下载"`
	Downloads           int64      `gorm:"default:0;comment:源站下载次数"`
	Likes               int64      `gorm:"default:0;comment:源站点赞次数"`
	SourceCreatedAt     *time.Time `gorm:"comment:源站创建时间"`
	SourceUpdatedAt     *time.Time `gorm:"comment:源站更新时间"`
	MetadataRefreshedAt *time.Time `gorm:"index;comment:源站元数据刷新时间"`
}

type ModelDatasetDiscoveryStatus string

const (
	ModelDatasetDiscoveryStatusDiscovered  ModelDatasetDiscoveryStatus = "discovered"
	ModelDatasetDiscoveryStatusRegistered  ModelDatasetDiscoveryStatus = "registered"
	ModelDatasetDiscoveryStatusPathMissing ModelDatasetDiscoveryStatus = "path_missing"
	ModelDatasetDiscoveryStatusMissing     ModelDatasetDiscoveryStatus = "missing"
	ModelDatasetDiscoveryStatusIgnored     ModelDatasetDiscoveryStatus = "ignored"
)

type ModelDatasetDiscoveryScope string

const (
	ModelDatasetDiscoveryScopePublic  ModelDatasetDiscoveryScope = "public"
	ModelDatasetDiscoveryScopeAccount ModelDatasetDiscoveryScope = "account"
	ModelDatasetDiscoveryScopeUser    ModelDatasetDiscoveryScope = "user"
)

type ModelDatasetDiscoveryEvidence struct {
	HasConfig            bool                 `json:"hasConfig"`
	HasReadme            bool                 `json:"hasReadme"`
	WeightFiles          int                  `json:"weightFiles"`
	MatchedFiles         []string             `json:"matchedFiles,omitempty"`
	Provider             ModelDatasetProvider `json:"provider,omitempty"`
	RepositoryID         string               `json:"repositoryId,omitempty"`
	RepositoryURL        string               `json:"repositoryUrl,omitempty"`
	ProvenanceSource     string               `json:"provenanceSource,omitempty"`
	ProvenanceConfidence string               `json:"provenanceConfidence,omitempty"`
	ConfigNameOrPath     string               `json:"configNameOrPath,omitempty"`
	CandidateURLs        []string             `json:"candidateUrls,omitempty"`
	FilesystemUID        string               `json:"filesystemUid,omitempty"`
	FilesystemGID        string               `json:"filesystemGid,omitempty"`
	ModifiedAt           *time.Time           `json:"modifiedAt,omitempty"`
	OwnerUserID          *uint                `json:"ownerUserId,omitempty"`
	OwnerUsername        string               `json:"ownerUsername,omitempty"`
}

// ModelDatasetDiscovery is a non-authoritative filesystem inventory record.
// Discovering a path never grants sharing permissions or publishes a Dataset.
type ModelDatasetDiscovery struct {
	gorm.Model
	DiscoveryKey string                                            `gorm:"type:varchar(1100);not null;uniqueIndex;comment:稳定发现键"`
	Path         string                                            `gorm:"type:varchar(1024);index;comment:文件系统路径,允许为空"`
	Scope        ModelDatasetDiscoveryScope                        `gorm:"type:varchar(32);not null;index"`
	ScopeID      *uint                                             `gorm:"index;comment:用户或队列ID"`
	DetectedType DataType                                          `gorm:"type:varchar(32);not null;index"`
	DetectedName string                                            `gorm:"type:varchar(256);not null"`
	Evidence     datatypes.JSONType[ModelDatasetDiscoveryEvidence] `gorm:"comment:文件系统检测依据"`
	SizeBytes    int64                                             `gorm:"not null;default:0"`
	DatasetID    *uint                                             `gorm:"index"`
	SourceID     *uint                                             `gorm:"index"`
	Status       ModelDatasetDiscoveryStatus                       `gorm:"type:varchar(32);not null;default:discovered;index"`
	FirstSeenAt  time.Time                                         `gorm:"not null"`
	LastSeenAt   time.Time                                         `gorm:"not null;index"`
}
