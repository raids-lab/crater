package utils

import (
	"fmt"
	"regexp"
	"time"

	// reference: https://stackoverflow.com/questions/60103251/time-loadlocation-works-regularly-but-throws-an-error-on-my-docker-instance-how
	_ "time/tzdata"

	"github.com/google/uuid"

	"github.com/raids-lab/crater/pkg/config"
)

const (
	splitLinkRegExp  = `^([^/]+)/([^/]+)/(.+):([^/:]+)$`
	getNameTagRegExp = `([^/]+/){2}([^:]+):([^/]+)$`
	splitLinkParts   = 5
	parts            = 4
)

func SplitImageLink(imageLink string) (ip, project, repository, tag string, err error) {
	re := regexp.MustCompile(splitLinkRegExp)
	matches := re.FindStringSubmatch(imageLink)
	if len(matches) != splitLinkParts {
		return "", "", "", "", fmt.Errorf("invalid image link: %s", imageLink)
	}
	ip, project, repository, tag = matches[1], matches[2], matches[3], matches[4]
	return ip, project, repository, tag, nil
}

func GetImageNameAndTag(imageLink string) (name, tag string, err error) {
	re := regexp.MustCompile(getNameTagRegExp)
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
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return "", err
	}
	now := time.Now().In(loc)
	imageTag := fmt.Sprintf("%02d%02d%02d%02d-%s", now.Month(), now.Day(), now.Hour(), now.Minute(), uuid.New().String()[:4])
	newImageLink = fmt.Sprintf("%s/%s/%s:%s", registryServer, registryProject, imageName, imageTag)
	return newImageLink, nil
}
