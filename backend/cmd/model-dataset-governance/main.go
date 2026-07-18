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
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/governance/modeldataset"
	"github.com/raids-lab/crater/pkg/config"
)

var (
	AppVersion = "unknown"
	CommitSHA  = "unknown"
	BuildType  = "development"
	BuildTime  = "unknown"
)

const (
	defaultStorageRoot       = "/crater"
	defaultModelsDirectory   = "Models"
	defaultDatasetsDirectory = "Datasets"
	defaultLogicalPrefix     = "public"
	defaultMaxDepth          = 8
	defaultReadmeBytes       = 64 * 1024
	defaultScanTimeout       = 30 * time.Minute
)

func main() {
	var (
		apply                = flag.Bool("apply", false, "write source links and discovery records; default is dry-run")
		storageRoot          = flag.String("storage-root", envOrDefault("CRATER_STORAGE_ROOT", defaultStorageRoot), "filesystem mount root")
		logicalPublicPrefix  = flag.String("logical-public-prefix", defaultLogicalPrefix, "logical public prefix used by download records")
		physicalPublicPrefix = flag.String("physical-public-prefix", "", "physical public prefix; defaults to storage.prefix.public")
		modelsDirectory      = flag.String("models-subdirectory", defaultModelsDirectory, "models subdirectory under the public prefix")
		modelsDirectories    = flag.String(
			"models-subdirectories", "",
			"optional comma-separated model subdirectories; takes precedence over --models-subdirectory",
		)
		datasetsDirectory   = flag.String("datasets-subdirectory", defaultDatasetsDirectory, "datasets subdirectory under the public prefix")
		maxDepth            = flag.Int("max-depth", defaultMaxDepth, "maximum scan depth below each resource root")
		excludedDirectories = flag.String(
			"exclude-directories",
			".cache,.git,.conda,node_modules,site-packages,test,tests,tmp,temp",
			"comma-separated directory basenames to skip",
		)
		weightPatterns = flag.String(
			"model-weight-patterns",
			"*.safetensors,pytorch_model*.bin,model*.bin,*.gguf,tf_model.h5,flax_model.msgpack",
			"comma-separated model weight filename patterns",
		)
		datasetMarkerPatterns = flag.String(
			"dataset-marker-patterns", "",
			"comma-separated dataset marker patterns; empty disables filesystem-only dataset discovery",
		)
		maxReadmeBytes = flag.Int("max-readme-bytes", defaultReadmeBytes, "maximum bytes read from a local README")
		scanTimeout    = flag.Duration("scan-timeout", defaultScanTimeout, "maximum filesystem scan duration")
		showVersion    = flag.Bool("version", false, "print build information and exit")
	)
	flag.Parse()
	if *showVersion {
		fmt.Printf("version=%s commit=%s build_type=%s build_time=%s\n", AppVersion, CommitSHA, BuildType, BuildTime)
		return
	}

	physicalPrefix := strings.TrimSpace(*physicalPublicPrefix)
	if physicalPrefix == "" {
		physicalPrefix = config.GetConfig().Storage.Prefix.Public
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *scanTimeout)
	defer cancel()

	candidates, err := modeldataset.ScanPublic(ctx, &modeldataset.ScanOptions{
		StorageRoot:           *storageRoot,
		PublicPrefix:          physicalPrefix,
		ModelsSubdirectory:    *modelsDirectory,
		ModelsSubdirectories:  splitCSV(*modelsDirectories),
		DatasetsSubdirectory:  *datasetsDirectory,
		MaxDepth:              *maxDepth,
		ExcludedDirectories:   splitCSV(*excludedDirectories),
		WeightPatterns:        splitCSV(*weightPatterns),
		DatasetMarkerPatterns: splitCSV(*datasetMarkerPatterns),
	})
	if err != nil {
		panic(fmt.Errorf("scan public model and dataset storage: %w", err))
	}

	report, err := modeldataset.ReconcilePublic(ctx, query.GetDB(), candidates, &modeldataset.ReconcileOptions{
		Apply:                 *apply,
		LogicalPublicPrefix:   *logicalPublicPrefix,
		PhysicalPublicPrefix:  physicalPrefix,
		PhysicalUserPrefix:    config.GetConfig().Storage.Prefix.User,
		PhysicalAccountPrefix: config.GetConfig().Storage.Prefix.Account,
		MaxReadmeBytes:        *maxReadmeBytes,
		Now:                   time.Now(),
	})
	if err != nil {
		panic(fmt.Errorf("reconcile public model and dataset storage: %w", err))
	}
	output := struct {
		Mode                 string                       `json:"mode"`
		StorageRoot          string                       `json:"storageRoot"`
		PhysicalPublicPrefix string                       `json:"physicalPublicPrefix"`
		Report               modeldataset.ReconcileReport `json:"report"`
	}{
		Mode:                 map[bool]string{true: "apply", false: "dry-run"}[*apply],
		StorageRoot:          *storageRoot,
		PhysicalPublicPrefix: physicalPrefix,
		Report:               report,
	}
	encoded, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(encoded))
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
