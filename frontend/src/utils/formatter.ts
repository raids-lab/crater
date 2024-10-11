/**
 * Calculates the time difference between the current date and a given date string.
 * @param dateString - The date string to calculate the difference from.
 * @returns A formatted string representing the time difference.
 */
export const getDateDiff = (dateString: string | undefined): string => {
  if (!dateString || dateString === "") {
    return "";
  }
  const now = new Date();
  const createdAt = new Date(dateString);
  const diff = now.getTime() - createdAt.getTime();

  let formatted = "";
  if (diff < 0) {
    formatted = "刚刚";
  } else if (diff > 1000 * 60 * 60 * 24 * 30 * 12) {
    const yearDiff = Math.floor(diff / (1000 * 60 * 60 * 24 * 30 * 12));
    formatted = `${yearDiff} 年前`;
  } else if (diff > 1000 * 60 * 60 * 24 * 30) {
    const monthDiff = Math.floor(diff / (1000 * 60 * 60 * 24 * 30));
    formatted = `${monthDiff} 个月前`;
  } else if (diff > 1000 * 60 * 60 * 24) {
    formatted = `${Math.floor(diff / (1000 * 60 * 60 * 24))} 天前`;
  } else if (diff > 1000 * 60 * 60) {
    formatted = `${Math.floor(diff / (1000 * 60 * 60))} 小时前`;
  } else if (diff > 1000 * 60) {
    formatted = `${Math.floor(diff / (1000 * 60))} 分钟前`;
  } else {
    formatted = `${Math.floor(diff / 1000)} 秒前`;
  }
  return formatted;
};

// crater-harbor.act.buaa.edu.cn/docker.io/jupyter/base-notebook:ubuntu-22.04 -> base-notebook:ubuntu-22.04
export const shortenImageName = (imageName: string): string => {
  return imageName.split("/").slice(-2).join("/").replace("docker.io/", "");
};

export const shortestImageName = (imageName: string): string => {
  return imageName.split("/").slice(-1).join("/");
};
