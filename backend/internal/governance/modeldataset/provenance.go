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
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/raids-lab/crater/dao/model"
)

const maxProvenanceFileBytes = 256 * 1024

const (
	provenanceConfidenceHigh   = "high"
	provenanceConfidenceMedium = "medium"
)

var (
	huggingFaceURLPattern = regexp.MustCompile(`https?://(?:www\.)?(?:huggingface\.co|hf-mirror\.com)/([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)`)
	modelScopeURLPattern  = regexp.MustCompile(`https?://(?:www\.)?modelscope\.cn/(?:models|datasets)/([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)`)
	huggingFaceSSHPattern = regexp.MustCompile(`(?:git@|ssh://git@)hf\.co[:/]([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)`)
)

type repositoryReference struct {
	Provider     model.ModelDatasetProvider
	RepositoryID string
	URL          string
}

func collectFilesystemEvidence(directory string, evidence *model.ModelDatasetDiscoveryEvidence) error {
	info, err := os.Stat(directory)
	if err != nil {
		return err
	}
	modifiedAt := info.ModTime()
	evidence.ModifiedAt = &modifiedAt
	evidence.FilesystemUID, evidence.FilesystemGID = filesystemOwnership(info)

	configNameOrPath, err := readConfigNameOrPath(filepath.Join(directory, "config.json"))
	if err != nil {
		return err
	}
	evidence.ConfigNameOrPath = configNameOrPath

	gitReferences, err := referencesFromFile(filepath.Join(directory, ".git", "config"))
	if err != nil {
		return err
	}
	if reference, ok := oneReference(gitReferences); ok {
		applyRepositoryEvidence(evidence, reference, "git_remote", provenanceConfidenceHigh)
		return nil
	}

	readmeReferences := make([]repositoryReference, 0)
	for _, name := range []string{"README.md", "readme.md", "README.MD", "README"} {
		references, readErr := referencesFromFile(filepath.Join(directory, name))
		if readErr != nil {
			return readErr
		}
		readmeReferences = append(readmeReferences, references...)
	}
	readmeReferences = uniqueReferences(readmeReferences)
	for _, reference := range readmeReferences {
		evidence.CandidateURLs = append(evidence.CandidateURLs, reference.URL)
	}
	if reference, ok := matchingReadmeReference(readmeReferences, filepath.Base(directory), configNameOrPath); ok {
		applyRepositoryEvidence(evidence, reference, "readme_url", provenanceConfidenceHigh)
		return nil
	}

	if validRepositoryID(configNameOrPath) {
		evidence.RepositoryID = strings.TrimSuffix(configNameOrPath, ".git")
		evidence.ProvenanceSource = "config_name_or_path"
		evidence.ProvenanceConfidence = provenanceConfidenceMedium
	}
	return nil
}

// filesystemOwnership uses the platform-provided stat object without tying the
// scanner to one operating system. UID/GID remain optional evidence on systems
// that do not expose Unix ownership.
func filesystemOwnership(info os.FileInfo) (uid, gid string) {
	stat := reflect.ValueOf(info.Sys())
	if !stat.IsValid() {
		return "", ""
	}
	if stat.Kind() == reflect.Pointer {
		stat = stat.Elem()
	}
	if !stat.IsValid() || stat.Kind() != reflect.Struct {
		return "", ""
	}
	value := func(name string) string {
		field := stat.FieldByName(name)
		if !field.IsValid() {
			return ""
		}
		switch field.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return strconv.FormatUint(field.Uint(), 10)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return strconv.FormatInt(field.Int(), 10)
		default:
			return ""
		}
	}
	return value("Uid"), value("Gid")
}

func readConfigNameOrPath(path string) (string, error) {
	data, err := readSmallFile(path)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", nil
	}
	var config struct {
		NameOrPath string `json:"_name_or_path"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		// A malformed config must not prevent inventorying an otherwise complete model.
		return "", nil
	}
	return strings.TrimSpace(config.NameOrPath), nil
}

func referencesFromFile(path string) ([]repositoryReference, error) {
	data, err := readSmallFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	return referencesFromText(string(data)), nil
}

func readSmallFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxProvenanceFileBytes))
	if err != nil {
		return nil, err
	}
	return data, nil
}

func referencesFromText(text string) []repositoryReference {
	result := make([]repositoryReference, 0)
	appendMatches := func(pattern *regexp.Regexp, provider model.ModelDatasetProvider, canonicalBase string) {
		for _, match := range pattern.FindAllStringSubmatch(text, -1) {
			repositoryID := strings.TrimSuffix(strings.TrimRight(match[1], ").,;]}'\""), ".git")
			result = append(result, repositoryReference{
				Provider: provider, RepositoryID: repositoryID, URL: canonicalBase + repositoryID,
			})
		}
	}
	appendMatches(huggingFaceURLPattern, model.ModelDatasetProviderHuggingFace, "https://huggingface.co/")
	appendMatches(huggingFaceSSHPattern, model.ModelDatasetProviderHuggingFace, "https://huggingface.co/")
	appendMatches(modelScopeURLPattern, model.ModelDatasetProviderModelScope, "https://modelscope.cn/models/")
	return uniqueReferences(result)
}

func uniqueReferences(references []repositoryReference) []repositoryReference {
	result := make([]repositoryReference, 0, len(references))
	seen := make(map[string]struct{})
	for _, reference := range references {
		key := string(reference.Provider) + "\x00" + reference.RepositoryID
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, reference)
	}
	return result
}

func oneReference(references []repositoryReference) (repositoryReference, bool) {
	references = uniqueReferences(references)
	returnReference := repositoryReference{}
	if len(references) == 1 {
		returnReference = references[0]
		return returnReference, true
	}
	return returnReference, false
}

func matchingReadmeReference(
	references []repositoryReference,
	directoryName, configNameOrPath string,
) (repositoryReference, bool) {
	matches := make([]repositoryReference, 0)
	for _, reference := range references {
		parts := strings.Split(reference.RepositoryID, "/")
		baseMatches := len(parts) == 2 && strings.EqualFold(parts[1], directoryName)
		configMatches := validRepositoryID(configNameOrPath) &&
			strings.EqualFold(strings.TrimSuffix(configNameOrPath, ".git"), reference.RepositoryID)
		if baseMatches || configMatches {
			matches = append(matches, reference)
		}
	}
	return oneReference(matches)
}

func validRepositoryID(value string) bool {
	value = strings.TrimSpace(strings.TrimSuffix(value, ".git"))
	if value == "" || filepath.IsAbs(value) || strings.Contains(value, "\\") || strings.Contains(value, ":") {
		return false
	}
	parts := strings.Split(value, "/")
	return len(parts) == 2 && parts[0] != "" && parts[1] != "" && parts[0] != "." && parts[0] != ".."
}

func applyRepositoryEvidence(
	evidence *model.ModelDatasetDiscoveryEvidence,
	reference repositoryReference,
	source, confidence string,
) {
	evidence.Provider = reference.Provider
	evidence.RepositoryID = reference.RepositoryID
	evidence.RepositoryURL = reference.URL
	evidence.ProvenanceSource = source
	evidence.ProvenanceConfidence = confidence
}
