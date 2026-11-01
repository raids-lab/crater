package handler

import (
	"slices"
	"strings"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/pkg/config"
)

// An explicit profile list replaces the built-in image, allowing private mirrors
// without silently admitting arbitrary engine versions or connector packages.
func servingProfileImages(id, engine string) []string {
	images, configured := config.GetConfig().Kthena.RuntimeImages[id]
	if !configured {
		images = []string{kthena.VLLMImage}
		if engine == kthena.EngineSGLang {
			images = []string{kthena.SGLangImage}
		}
	}
	allowed := []string{}
	for _, image := range images {
		if pinnedServingImage(image) {
			allowed = append(allowed, image)
		}
	}
	return allowed
}

func pinnedServingImage(image string) bool {
	if strings.TrimSpace(image) != image || strings.ContainsAny(image, " \t\n") {
		return false
	}
	if _, digest, ok := strings.Cut(image, "@sha256:"); ok {
		const sha256HexLength = 64
		if len(digest) != sha256HexLength {
			return false
		}
		for _, c := range digest {
			if !strings.ContainsRune("0123456789abcdef", c) {
				return false
			}
		}
		return true
	}
	leaf := image[strings.LastIndex(image, "/")+1:]
	_, tag, ok := strings.Cut(leaf, ":")
	return ok && tag != "" && tag != "latest"
}

func servingProfileConnector(engine, layout string) string {
	if layout != kthena.LayoutPD {
		return ""
	}
	if engine == kthena.EngineVLLM {
		return "NixlConnector"
	}
	return "mooncake"
}

func validateServingProfileImages(role *kthena.RoleSpec, images []string) error {
	if !slices.Contains(images, role.Entry.Image) || (role.Worker != nil && !slices.Contains(images, role.Worker.Image)) {
		return bizerr.BadRequest.ParameterError.New("role " + role.Name + ": image is not approved for this serving profile")
	}
	return nil
}
