package imageregistry

import (
	"context"
)

type ImageRegistryInterface interface {
	/// Project operations

	// CheckOrCreateProjectForUser checks if the project exists for the user, if not, create one.
	CheckOrCreateProjectForUser(ctx context.Context, username string) error

	// GetQuotaForProject gets the quota size for the project.
	UpdateQuotaForProject(ctx context.Context, projectName string, quotaSize int64) error

	/// Image operations

	// DeleteImageFromProject deletes the image from the project.
	DeleteImageFromProject(ctx context.Context, fullImageURL string) error

	// GetImageSize gets the size of the image.
	GetImageSize(ctx context.Context, fullImageName string) (int64, error)

	CheckOrCreateUser(ctx context.Context, username string) (string, error)

	CheckUserExist(ctx context.Context, username string) bool

	AddProjectMember(c context.Context, username string) error

	CreateUser(c context.Context, username string) (string, error)

	DeleteUser(c context.Context, username string) error
}

type ImageRegistry struct {
	harborClient *HarborClient
}

func NewImageRegistry() ImageRegistryInterface {
	harborClient := NewHarborClient()
	return &ImageRegistry{
		harborClient: &harborClient,
	}
}
