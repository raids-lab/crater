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

func GenerateNewImageLinkForEnvdBuild(username, python, base, imageName, imageTag string) (newImageLink string, err error) {
	registryServer := config.GetConfig().ImageRegistry.Server
	registryProject := fmt.Sprintf("user-%s", username)
	if imageName == "" {
		imageName = "envd"
	}
	if imageTag == "" {
		if base == "" {
			imageTag = fmt.Sprintf("py%s-%s", python, uuid.New().String()[:4])
		} else {
			imageTag = fmt.Sprintf("py%s-%s-%s", python, base, uuid.New().String()[:4])
		}
	}
	newImageLink = fmt.Sprintf("%s/%s/%s:%s", registryServer, registryProject, imageName, imageTag)
	return newImageLink, nil
}

func GenerateNewImageLinkForDockerfileBuild(imageLink, username, imageName, imageTag string) (newImageLink string, err error) {
	registryServer := config.GetConfig().ImageRegistry.Server
	registryProject := fmt.Sprintf("user-%s", username)
	loc, _ := time.LoadLocation("Asia/Shanghai")
	if imageName == "" {
		if imageName, _, err = GetImageNameAndTag(imageLink); err != nil {
			return "", err
		}
	}
	if imageTag == "" {
		now := time.Now().In(loc)
		imageTag = fmt.Sprintf("%02d%02d%02d%02d-%s", now.Month(), now.Day(), now.Hour(), now.Minute(), uuid.New().String()[:4])
	}
	newImageLink = fmt.Sprintf("%s/%s/%s:%s", registryServer, registryProject, imageName, imageTag)
	return newImageLink, nil
}
