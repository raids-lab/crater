package utils

import (
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/raids-lab/crater/pkg/config"
)

const (
	imageLinkRegExp = `([^/]+/){2}([^:]+):([^/]+)$`
	parts           = 4
)

func GetImageNameAndTag(imageLink string) (name, tag string, err error) {
	re := regexp.MustCompile(imageLinkRegExp)
	matches := re.FindStringSubmatch(imageLink)
	if len(matches) != parts {
		return "", "", fmt.Errorf("invalid image link: %s", imageLink)
	}
	name, tag = matches[2], matches[3]
	return name, tag, nil
}

func GenerateNewImageLink(imageLink, username string) (newImageLink string, err error) {
	imageName, _, err := GetImageNameAndTag(imageLink)
	if err != nil {
		return "", err
	}
	registryServer := config.GetConfig().ACT.Image.RegistryServer
	registryProject := fmt.Sprintf("user-%s", username)
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now().In(loc)
	imageTag := fmt.Sprintf("%d%d-%d%d-%s", now.Month(), now.Day(), now.Hour(), now.Minute(), uuid.New().String()[:4])
	newImageLink = fmt.Sprintf("%s/%s/%s:%s", registryServer, registryProject, imageName, imageTag)
	return newImageLink, nil
}
