package imageregistry

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math"
	"regexp"
	"strings"

	harbormodelv2 "github.com/mittwald/goharbor-client/v5/apiv2/model"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/logutils"
)

var (
	ProjectIsPublic = true
	//nolint:mnd // default project quota: 40GB
	DefaultQuotaSize = int64(40 * math.Pow(2, 30))
	TmpEmailSuffix   = "@example.com"
	PasswordLength   = 20
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

// GenerateRandomPassword generates a random 10-character password
func GenerateRandomPassword(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Encode the bytes to a base64 string and take the first 10 characters
	password := base64.URLEncoding.EncodeToString(bytes)[:length]
	return password, nil
}

func (r *ImageRegistry) CheckUserExist(c context.Context, username string) bool {
	name := intstr.IntOrString{
		Type:   intstr.String,
		StrVal: username,
	}
	if exist, _ := r.harborClient.UserExists(c, name); exist {
		return true
	}
	return false
}

func (r *ImageRegistry) CreateUser(c context.Context, username string) (string, error) {
	email := username + TmpEmailSuffix
	password, err := GenerateRandomPassword(PasswordLength)
	if err != nil {
		logutils.Log.Errorf("generate random password failed! err:%+v", err)
		return "", err
	}
	if err = r.harborClient.NewUser(c, username, email, username, password, ""); err != nil {
		logutils.Log.Errorf("create harbor user failed! err:%+v", err)
		return "", err
	}
	return password, nil
}

func (r *ImageRegistry) AddProjectMember(c context.Context, username string) error {
	projectName := fmt.Sprintf("user-%s", username)
	harborMember := &harbormodelv2.ProjectMember{
		RoleID: 1,
		MemberUser: &harbormodelv2.UserEntity{
			Username: username,
		},
	}
	return r.harborClient.AddProjectMember(c, projectName, harborMember)
}

func (r *ImageRegistry) CheckOrCreateUser(c context.Context, username string) (string, error) {
	if exist := r.CheckUserExist(c, username); exist {
		return "", nil
	}
	password, err := r.CreateUser(c, username)
	if err != nil {
		return "", err
	}

	if err = r.AddProjectMember(c, username); err != nil {
		logutils.Log.Errorf("add project member failed! err:%+v", err)
		return password, err
	}

	return password, nil
}

func (r *ImageRegistry) DeleteUser(c context.Context, username string) error {
	userResp, err := r.harborClient.GetUserByName(c, username)
	if err != nil {
		logutils.Log.Errorf("get harbor user failed! err:%+v", err)
		return err
	}
	return r.harborClient.DeleteUser(c, userResp.UserID)
}
