// src/atoms/configAtom.ts
import { atom } from "jotai";
import { loadable } from "jotai/utils";
import { logger } from "../loglevel";
import { AppConfig } from "./type";

// 创建异步原子来加载配置
export const configAtom = atom(async () => {
  let config: AppConfig;

  // 首先尝试从 config.json 读取配置
  try {
    const response = await fetch("/config.json");
    if (!response.ok) {
      throw new Error("Failed to load config");
    }
    config = await response.json();
  } catch (error) {
    logger.error("Error loading config from config.json:", error);
    // 如果无法加载，创建一个空的基础配置
    config = {
      version: "",
      url: { host: "", apiBase: "", apiVersion: "", websiteBase: "" },
      grafana: {
        overview: { main: "", schedule: "", network: "" },
        node: { basic: "", nvidia: "" },
        job: { basic: "", nvidia: "", pod: "" },
        user: { nvidia: "" },
      },
      featureFlags: {
        enableRegister: false,
        setDocsAsHome: false,
        enableMockServiceWorker: false,
      },
    } as AppConfig;
  }

  // 如果是开发环境，用环境变量覆盖部分配置
  if (import.meta.env.MODE === "development") {
    config = {
      ...config,
      version: import.meta.env.VITE_APP_VERSION,
      url: {
        ...config.url,
        host: import.meta.env.VITE_HOST,
        apiBase: import.meta.env.VITE_API_BASE_URL,
        apiVersion: import.meta.env.VITE_API_VERSION,
        websiteBase: import.meta.env.VITE_WEBSITE_BASE_URL,
      },
      grafana: {
        ...config.grafana,
        overview: {
          ...config.grafana.overview,
          main: import.meta.env.VITE_GRAFANA_OVERVIEW,
          schedule: import.meta.env.VITE_GRAFANA_SCHEDULE,
          network: import.meta.env.VITE_GRAFANA_NETWORK,
        },
        node: {
          ...config.grafana.node,
          basic: import.meta.env.VITE_GRAFANA_NODE,
          nvidia: import.meta.env.VITE_GRAFANA_GPU_DASHBOARD,
        },
        job: {
          ...config.grafana.job,
          basic: import.meta.env.VITE_GRAFANA_JOB_MONITOR,
          nvidia: import.meta.env.VITE_GRAFANA_JOB_GPU_MONITOR,
          pod: import.meta.env.VITE_GRAFANA_POD_MONITOR,
        },
        user: {
          ...config.grafana.user,
          nvidia: import.meta.env.VITE_GRAFANA_USER_GPU_DASHBOARD,
        },
      },
      featureFlags: {
        ...config.featureFlags,
        enableRegister: config.featureFlags.enableRegister,
        setDocsAsHome: import.meta.env.VITE_SET_DOCS_AS_HOME === "true",
        enableMockServiceWorker: import.meta.env.VITE_USE_MSW === "true",
      },
    } as AppConfig;
  }

  return config;
});

// 创建一个可加载版本以便在组件中处理加载状态
export const loadableConfigAtom = loadable(configAtom);

// 派生原子
export const asyncVersionAtom = atom(
  async (get) => (await get(configAtom)).url.apiVersion,
);

export const asyncAppVersionAtom = atom(
  async (get) => (await get(configAtom)).version,
);

export const asyncUrlHostAtom = atom(
  async (get) => (await get(configAtom)).url.host,
);

export const asyncUrlApiBaseAtom = atom(
  async (get) => (await get(configAtom)).url.apiBase,
);

export const asyncUrlWebsiteBaseAtom = atom(
  async (get) => (await get(configAtom)).url.websiteBase,
);

export const asyncDocsAsHomeAtom = atom(
  async (get) => (await get(configAtom)).featureFlags.setDocsAsHome,
);

// 派生原子 - 按类别导出
export const asyncGrafanaOverviewAtom = atom(
  async (get) => (await get(configAtom)).grafana.overview,
);
export const asyncGrafanaNodeAtom = atom(
  async (get) => (await get(configAtom)).grafana.node,
);
export const asyncGrafanaJobAtom = atom(
  async (get) => (await get(configAtom)).grafana.job,
);
export const asyncGrafanaUserAtom = atom(
  async (get) => (await get(configAtom)).grafana.user,
);

// Feature Flags
export const asyncFeatureFlags = atom(
  async (get) => (await get(configAtom)).featureFlags,
);
