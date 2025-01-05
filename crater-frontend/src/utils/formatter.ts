// crater-harbor.act.buaa.edu.cn/docker.io/jupyter/base-notebook:ubuntu-22.04 -> base-notebook:ubuntu-22.04
export const shortenImageName = (imageName: string): string => {
  return imageName.split("/").slice(-2).join("/").replace("docker.io/", "");
};

export const shortestImageName = (imageName: string): string => {
  return imageName.split("/").slice(-1).join("/");
};

export function formatBytes(bytes: number, decimals: number = 2): string {
  if (bytes === 0) return "0 B";
  if (bytes < 0) return "-" + formatBytes(-bytes, decimals);

  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];

  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const value = bytes / Math.pow(k, i);

  return `${value.toFixed(decimals)} ${sizes[i]}`;
}
