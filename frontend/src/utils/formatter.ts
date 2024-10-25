// crater-harbor.act.buaa.edu.cn/docker.io/jupyter/base-notebook:ubuntu-22.04 -> base-notebook:ubuntu-22.04
export const shortenImageName = (imageName: string): string => {
  return imageName.split("/").slice(-2).join("/").replace("docker.io/", "");
};

export const shortestImageName = (imageName: string): string => {
  return imageName.split("/").slice(-1).join("/");
};
