package imageregistry

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"

	harbormodelv2 "github.com/mittwald/goharbor-client/v5/apiv2/model"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/logutils"
)

var (
	ProjectIsPublic = true
	//nolint:mnd // default project quota: 40GB
	DefaultQuotaSize = int64(40 * math.Pow(2, 30))
)

// CheckOrCreateProjectForUser checks if the project for the user exists, if not, create a new project for the user.
func (r *ImageRegistry) CheckOrCreateProjectForUser(c context.Context, username string) error {
	projectName := fmt.Sprintf("user-%s", username)
	if exist, _ := r.harborClient.ProjectExists(c, projectName); exist {
		return nil
	}

	u := query.User
	if _, err := u.WithContext(c).Where(u.Name.Eq(username)).
		Update(u.ImageQuota, DefaultQuotaSize); err != nil {
		logutils.Log.Errorf("save user imageQuota failed, err:%v", err)
		return err
	}

	if err := r.harborClient.NewProject(c, &harbormodelv2.ProjectReq{
		ProjectName:  projectName,
		Public:       &ProjectIsPublic,
		StorageLimit: &DefaultQuotaSize,
	}); err != nil {
		logutils.Log.Errorf("create harbor project failed! err:%+v", err)
		return err
	}

	return nil
}

func (r *ImageRegistry) getImageInfo(fullImageURL string) (projectName, imageName, imageTag string, err error) {
	// fullImageURL like: ***REMOVED***-images/crater:latest
	// projectName: crater-images, imageName: crater, imageTag: latest
	// get projectName, imageName, imageTag from fullImageURL by regex
	// check if full image url starts with inner registry
	if !strings.HasPrefix(fullImageURL, r.harborClient.AuthInfo.RegistryServer) {
		// skip delete if image is not in inner registry
		return "", "", "", fmt.Errorf("image is not in inner registry: %s", fullImageURL)
	}

	regexPattern := fmt.Sprintf(`%s/(.*?)/(.*?):(.*?)$`, r.harborClient.AuthInfo.RegistryServer)
	re := regexp.MustCompile(regexPattern)
	matches := re.FindStringSubmatch(fullImageURL)
	exceptedMatchesLen := 4
	if len(matches) != exceptedMatchesLen {
		logutils.Log.Errorf("invalid full image url: %s", fullImageURL)
		return "", "", "", fmt.Errorf("invalid full image url: %s", fullImageURL)
	}
	projectName = matches[1]
	imageName = matches[2]
	imageTag = matches[3]
	return projectName, imageName, imageTag, nil
}

// DeleteImageFromProject deletes the image from the project.
func (r *ImageRegistry) DeleteImageFromProject(c context.Context, fullImageURL string) error {
	projectName, imageName, imageTag, err := r.getImageInfo(fullImageURL)
	if err != nil {
		return err
	}

	return r.harborClient.DeleteArtifact(c, projectName, imageName, imageTag)
}

func (r *ImageRegistry) UpdateQuotaForProject(c context.Context, projectName string, quotaSize int64) error {
	project, err := r.harborClient.GetProject(c, projectName)
	if err != nil {
		logutils.Log.Errorf("get harbor project failed, err: %+v", err)
		return err
	}
	return r.harborClient.UpdateStorageQuotaByProjectID(c, int64(project.ProjectID), quotaSize)
}

func (r *ImageRegistry) GetImageSize(c context.Context, fullImageName string) (int64, error) {
	projectName, imageName, imageTag, err := r.getImageInfo(fullImageName)
	if err != nil {
		return 0, err
	}

	imageArtifact, err := r.harborClient.GetArtifact(c, projectName, imageName, imageTag)
	if err != nil {
		logutils.Log.Errorf("get image artifact failed! err:%+v", err)
		return 0, err
	}
	return imageArtifact.Size, nil
}
