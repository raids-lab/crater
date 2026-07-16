package handler

import (
	"testing"

	"github.com/raids-lab/crater/dao/model"
)

func TestNormalizedOrganizationLogoKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		organization string
		repositoryID string
		want         string
	}{
		{name: "explicit organization", organization: " Qwen ", repositoryID: "ignored", want: "qwen"},
		{name: "legacy repository", repositoryID: "qwen/qwen2-0.5b", want: "qwen"},
		{name: "manual resource name", repositoryID: "qwen2-0.5b", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizedOrganizationLogoKey(tt.organization, tt.repositoryID); got != tt.want {
				t.Fatalf("normalizedOrganizationLogoKey(%q, %q) = %q, want %q", tt.organization, tt.repositoryID, got, tt.want)
			}
		})
	}
}

func TestDeduplicateDatasets(t *testing.T) {
	t.Parallel()

	newerDuplicate := &model.Dataset{Name: "SKYLENAGE/SkyJM-Gen-4B", Type: model.DataTypeModel}
	newerDuplicate.ID = 92
	original := &model.Dataset{Name: "skylenage/SkyJM-Gen-4B", Type: model.DataTypeModel}
	original.ID = 91
	datasetWithSameName := &model.Dataset{Name: "SKYLENAGE/SkyJM-Gen-4B", Type: model.DataTypeDataset}
	datasetWithSameName.ID = 93

	downloadedKeys := map[string]struct{}{
		resourceMetadataKey("skylenage/skyjm-gen-4b", string(model.DataTypeModel)): {},
	}
	got := deduplicateDownloadedDatasets(
		[]*model.Dataset{newerDuplicate, original, datasetWithSameName}, downloadedKeys,
	)
	if len(got) != 2 {
		t.Fatalf("deduplicateDatasets() returned %d rows, want 2", len(got))
	}
	if got[0].ID != original.ID {
		t.Fatalf("deduplicateDatasets() kept ID %d, want canonical ID %d", got[0].ID, original.ID)
	}
	if got[1].ID != datasetWithSameName.ID {
		t.Fatalf("deduplicateDatasets() incorrectly merged different resource types")
	}
}

func TestDeduplicateDownloadedDatasetsPreservesUserResources(t *testing.T) {
	t.Parallel()

	first := &model.Dataset{Name: "experiment", Type: model.DataTypeDataset}
	first.ID = 1
	second := &model.Dataset{Name: "experiment", Type: model.DataTypeDataset}
	second.ID = 2

	got := deduplicateDownloadedDatasets([]*model.Dataset{first, second}, nil)
	if len(got) != 2 {
		t.Fatalf("deduplicateDownloadedDatasets() merged user resources, got %d rows", len(got))
	}
}
