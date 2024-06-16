import { atom, createStore } from "jotai";
import { atomWithStorage, useResetAtom } from "jotai/utils";
import { UserInfo } from "@/hooks/useAuth";
import { AccessMode, Role, CurrentAccount } from "@/services/api/auth";

export const store = createStore();

/**
 * LocalStorage and Jotai Keys
 */
const USER_INFO_KEY = "user_info";
const LAST_VIEW_KEY = "last_view";
const CURRENT_ACCOUNT_KEY = "current_account";
const SETTINGS_KEY = "settings";
export const ACCESS_TOKEN_KEY = "access_token";
export const REFRESH_TOKEN_KEY = "refresh_token";
export const VITE_UI_THEME_KEY = "vite_ui_theme";

/**
 * Navigation BreadCrumb
 */
export type BreadCrumbItem = {
  title: string;
  path?: string;
};
export const globalBreadCrumb = atom([] as BreadCrumbItem[]);

/**
 * User Info
 */
const defaultUserContext: UserInfo = {
  name: "",
};

export const globalUserInfo = atomWithStorage(
  USER_INFO_KEY,
  defaultUserContext,
  undefined,
  {
    getOnInit: true,
  },
);

/**
 * Remember the last view.
 * Will not be cleared when logout.
 */
export const globalLastView = atomWithStorage(LAST_VIEW_KEY, "", undefined, {
  getOnInit: true,
});

export const defaultAccount: CurrentAccount = {
  queue: "",
  roleQueue: Role.Guest,
  rolePlatform: Role.Guest,
  accessQueue: AccessMode.NotAllowed,
  accessPublic: AccessMode.NotAllowed,
};

// Hydrating from Local Storage on First Render: https://github.com/pmndrs/jotai/discussions/1737
export const globalAccount = atomWithStorage(
  CURRENT_ACCOUNT_KEY,
  defaultAccount,
  undefined,
  {
    getOnInit: true,
  },
);

export const globalSettings = atomWithStorage(
  SETTINGS_KEY,
  {
    scheduler: "volcano",
  },
  undefined,
  {
    getOnInit: true,
  },
);

export const globalJobUrl = atom((get) => {
  const scheduler = get(globalSettings).scheduler;
  switch (scheduler) {
    case "volcano":
      return "vcjobs";
    case "colocate":
      return "aijobs";
    case "sparse":
      return "spjobs";
    default:
      return "vcjobs";
  }
});

/**
 * Reset all states
 */
export const useResetStore = () => {
  const resetUserInfo = useResetAtom(globalUserInfo);
  const resetLastView = useResetAtom(globalLastView);
  const resetProject = useResetAtom(globalAccount);
  const resetSettings = useResetAtom(globalSettings);

  const resetAll = () => {
    // Jotai
    resetUserInfo();
    resetLastView();
    resetProject();
    resetSettings();
  };

  return { resetAll };
};
