import { adminPath } from "@/pages/Admin/path";
import { craterPath } from "@/pages/Portal/path";
import { recommendPath } from "@/pages/PortalR/path";

export type PathInfo = {
  path: string; // router path
  title: string; // title in sidebar
  titleNav?: string; // title in navibar (if not set, use title)
  children?: PathInfo[]; // children path info
};

const pathDict: PathInfo[] = [craterPath, recommendPath, adminPath];

/**
 *
 * @param path example: ['portal', 'job', 'ai']
 */
export const getBreadcrumbByPath = (
  path: string[],
): { path: string; title: string }[] | null => {
  const result = [];
  let currentPath = pathDict;
  for (let i = 0; i < path.length; i++) {
    const item = currentPath.find((item) => item.path === path[i]);
    if (item) {
      result.push({ title: item.titleNav ?? item.title, path: item.path });
      currentPath = item.children || [];
    } else {
      break;
    }
  }
  return result;
};

export const getTitleByPath = (path: string[]): string => {
  let currentPath = pathDict;
  for (let i = 0; i < path.length; i++) {
    const item = currentPath.find((item) => item.path === path[i]);
    if (item) {
      if (i === path.length - 1) {
        return item.title;
      }
      currentPath = item.children || [];
    } else {
      break;
    }
  }
  return "";
};
