package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/util"
)

type agentAccessibleImage struct {
	Image       *model.Image
	ShareStatus model.ImageShareType
}

func (mgr *AgentMgr) listAccessibleImages(ctx context.Context, token util.JWTMessage) ([]agentAccessibleImage, error) {
	imageQuery := query.Image
	results := make([]agentAccessibleImage, 0)
	seen := make(map[uint]struct{})
	appendUnique := func(images []*model.Image, shareStatus model.ImageShareType) {
		for _, image := range images {
			if image == nil || image.ID == 0 || image.ImageLink == "" {
				continue
			}
			if _, ok := seen[image.ID]; ok {
				continue
			}
			seen[image.ID] = struct{}{}
			results = append(results, agentAccessibleImage{
				Image:       image,
				ShareStatus: shareStatus,
			})
		}
	}

	oldPublicImages, err := imageQuery.WithContext(ctx).
		Preload(imageQuery.User).
		Where(imageQuery.IsPublic.Is(true)).
		Order(imageQuery.CreatedAt.Desc()).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list public images: %w", err)
	}
	appendUnique(oldPublicImages, model.Public)

	imageAccountQuery := query.ImageAccount
	newPublicShares, err := imageAccountQuery.WithContext(ctx).
		Preload(imageAccountQuery.Image).
		Preload(imageAccountQuery.Image.User).
		Where(imageAccountQuery.AccountID.Eq(model.DefaultAccountID)).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list shared public images: %w", err)
	}
	newPublicImages := make([]*model.Image, 0, len(newPublicShares))
	for _, share := range newPublicShares {
		newPublicImages = append(newPublicImages, &share.Image)
	}
	appendUnique(newPublicImages, model.Public)

	accountShares, err := imageAccountQuery.WithContext(ctx).
		Preload(imageAccountQuery.Image).
		Preload(imageAccountQuery.Image.User).
		Where(imageAccountQuery.AccountID.Eq(token.AccountID)).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list account images: %w", err)
	}
	accountImages := make([]*model.Image, 0, len(accountShares))
	for _, share := range accountShares {
		accountImages = append(accountImages, &share.Image)
	}
	appendUnique(accountImages, model.AccountShare)

	privateImages, err := imageQuery.WithContext(ctx).
		Preload(imageQuery.User).
		Where(imageQuery.UserID.Eq(token.UserID)).
		Order(imageQuery.CreatedAt.Desc()).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list private images: %w", err)
	}
	appendUnique(privateImages, model.Private)

	imageUserQuery := query.ImageUser
	userShares, err := imageUserQuery.WithContext(ctx).
		Preload(imageUserQuery.Image).
		Preload(imageUserQuery.Image.User).
		Where(imageUserQuery.UserID.Eq(token.UserID)).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list user-shared images: %w", err)
	}
	userImages := make([]*model.Image, 0, len(userShares))
	for _, share := range userShares {
		userImages = append(userImages, &share.Image)
	}
	appendUnique(userImages, model.UserShare)

	return results, nil
}

func buildAgentImageSummary(item agentAccessibleImage) map[string]any {
	description := ""
	if item.Image.Description != nil {
		description = strings.TrimSpace(*item.Image.Description)
	}
	imagePackName := ""
	if item.Image.ImagePackName != nil {
		imagePackName = *item.Image.ImagePackName
	}

	archs := item.Image.Archs.Data()
	if len(archs) == 0 {
		archs = []string{"linux/amd64"}
	}

	return map[string]any{
		"id":            item.Image.ID,
		"imageLink":     item.Image.ImageLink,
		"description":   description,
		"taskType":      item.Image.TaskType,
		"shareStatus":   item.ShareStatus,
		"imageSource":   item.Image.ImageSource.String(),
		"tags":          item.Image.Tags.Data(),
		"archs":         archs,
		"imagePackName": imagePackName,
		"createdAt":     item.Image.CreatedAt,
		"owner": map[string]any{
			"userID":   item.Image.User.ID,
			"username": item.Image.User.Name,
			"nickname": item.Image.User.Nickname,
		},
	}
}

func matchesImageJobType(taskType model.JobType, requested string) bool {
	requested = strings.TrimSpace(strings.ToLower(requested))
	if requested == "" || requested == string(model.JobTypeAll) {
		return true
	}
	switch requested {
	case "training":
		switch taskType {
		case model.JobTypeCustom, model.JobTypePytorch, model.JobTypeTensorflow, model.JobTypeDeepSpeed, model.JobTypeOpenMPI:
			return true
		default:
			return false
		}
	default:
		return strings.EqualFold(string(taskType), requested)
	}
}

func matchesImageKeyword(image *model.Image, keyword string) bool {
	keyword = strings.TrimSpace(strings.ToLower(keyword))
	if keyword == "" {
		return true
	}
	text := strings.ToLower(image.ImageLink)
	if image.Description != nil {
		text += " " + strings.ToLower(*image.Description)
	}
	if image.ImagePackName != nil {
		text += " " + strings.ToLower(*image.ImagePackName)
	}
	for _, tag := range image.Tags.Data() {
		text += " " + strings.ToLower(tag)
	}
	return strings.Contains(text, keyword)
}

func buildTrainingImageKeywords(taskDescription, framework string) []string {
	text := strings.ToLower(strings.TrimSpace(taskDescription + " " + framework))
	keywords := []string{}
	add := func(values ...string) {
		for _, value := range values {
			if value == "" {
				continue
			}
			already := false
			for _, existing := range keywords {
				if existing == value {
					already = true
					break
				}
			}
			if !already {
				keywords = append(keywords, value)
			}
		}
	}

	if strings.Contains(text, "pytorch") || strings.Contains(text, "torch") {
		add("pytorch", "torch")
	}
	if strings.Contains(text, "tensorflow") || strings.Contains(text, "tf") {
		add("tensorflow", "tf")
	}
	if strings.Contains(text, "意图") || strings.Contains(text, "nlp") || strings.Contains(text, "文本") ||
		strings.Contains(text, "分类") || strings.Contains(text, "bert") || strings.Contains(text, "transformer") {
		add("transformers", "bert", "nlp", "pytorch", "torch")
	}
	if strings.Contains(text, "jupyter") {
		add("jupyter")
	}
	if len(keywords) == 0 {
		add("python", "envd", "conda")
	}
	return keywords
}

func scoreTrainingImage(item agentAccessibleImage, keywords []string) (int, []string) {
	text := strings.ToLower(item.Image.ImageLink)
	if item.Image.Description != nil {
		text += " " + strings.ToLower(*item.Image.Description)
	}
	if item.Image.ImagePackName != nil {
		text += " " + strings.ToLower(*item.Image.ImagePackName)
	}
	for _, tag := range item.Image.Tags.Data() {
		text += " " + strings.ToLower(tag)
	}

	score := 0
	reasons := make([]string, 0, 4)
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			score += 3
			reasons = append(reasons, fmt.Sprintf("命中关键词 %s", keyword))
		}
	}
	switch item.Image.TaskType {
	case model.JobTypePytorch, model.JobTypeCustom:
		score += 2
		reasons = append(reasons, fmt.Sprintf("任务类型为 %s，适合作为训练镜像", item.Image.TaskType))
	case model.JobTypeJupyter:
		score += 1
		reasons = append(reasons, "适合作为交互式实验镜像")
	}
	switch item.ShareStatus {
	case model.Public, model.AccountShare:
		score++
		reasons = append(reasons, "当前账户可直接复用")
	}
	if item.Image.Description != nil && strings.TrimSpace(*item.Image.Description) != "" {
		score++
	}
	return score, uniqueStrings(reasons)
}

func recommendationConfidence(score int) string {
	switch {
	case score >= 8:
		return "high"
	case score >= 5:
		return "medium"
	default:
		return "low"
	}
}
