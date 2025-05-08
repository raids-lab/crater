import { adminPath } from "@/pages/Admin/path";
import { craterPath } from "@/pages/Portal/path";

export type PathInfo = {
  path: string; // router path
  titleKey: string; // i18n key for title
  titleNavKey?: string; // i18n key for titleNav
  isEmpty?: boolean; // if true, will disable the return in breadcrumb
  children?: PathInfo[]; // children path info
};

const pathDict: PathInfo[] = [craterPath, adminPath];

/**
 *
 * @param path example: ['portal', 'job', 'ai']
 */
export const getBreadcrumbByPath = (
  path: string[],
): { path: string; title: string; isEmpty?: boolean }[] | null => {
  const result = [];
  let currentPath = pathDict;
  for (let i = 0; i < path.length; i++) {
    const item = currentPath.find((item) => item.path === path[i]);
    if (item) {
      result.push({
        title: item.titleNavKey ?? item.titleKey,
        path: item.path,
        isEmpty: item.isEmpty,
      });
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
        return item.titleKey;
      }
      currentPath = item.children || [];
    } else {
      break;
    }
  }
  return "";
};
