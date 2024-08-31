package config

type ShareDirs []ShareDir

type ShareDir struct {
	Pvc       string `yaml:"pvc" json:"pvc"`
	Namespace string `yaml:"namespace" json:"namespace"`
}

var shareDirs ShareDirs = []ShareDir{
	{
		Pvc:       "dnn-train-data",
		Namespace: "crater-workspace",
	},
	{
		Pvc:       "jupyterhub-shared-volume",
		Namespace: "crater-workspace",
	},
}

func GetShareDirs() ShareDirs {
	return shareDirs
}

func GetShareDirNames() []string {
	var names []string
	for _, shareDir := range shareDirs {
		names = append(names, shareDir.Pvc)
	}
	return names
}
