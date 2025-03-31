// src/atoms/configAtom.ts
import { atom } from "jotai";
import { loadable } from "jotai/utils";
import { logger } from "../loglevel";
import { AppConfig } from "./type";

// 创建异步原子来加载配置
export const configAtom = atom(async () => {
  if (import.meta.env.MODE === "development") {
    // 开发环境直接从环境变量读取
    return {
      version: import.meta.env.VITE_APP_VERSION,
      url: {
        host: import.meta.env.VITE_HOST,
        apiBase: import.meta.env.VITE_API_BASE_URL,
        apiVersion: import.meta.env.VITE_API_VERSION,
        websiteBase: import.meta.env.VITE_WEBSITE_BASE_URL,
      },
      grafana: {
        overview: {
          main: import.meta.env.VITE_GRAFANA_OVERVIEW,
          schedule: import.meta.env.VITE_GRAFANA_SCHEDULE,
          network: import.meta.env.VITE_GRAFANA_NETWORK,
        },
        node: {
          basic: import.meta.env.VITE_GRAFANA_NODE,
          nvidia: import.meta.env.VITE_GRAFANA_GPU_DASHBOARD,
        },
        job: {
          basic: import.meta.env.VITE_GRAFANA_JOB_MONITOR,
          nvidia: import.meta.env.VITE_GRAFANA_JOB_GPU_MONITOR,
          pod: import.meta.env.VITE_GRAFANA_POD_MONITOR,
        },
      },
      featureFlags: {
        enableRegister: false,
        enableMockServiceWorker: import.meta.env.VITE_USE_MSW === "true",
      },
    } as AppConfig;
  } else {
    // 生产环境从配置文件读取
    try {
      const response = await fetch("/config.json");
      if (!response.ok) {
        throw new Error("Failed to load config");
      }
      const config: AppConfig = await response.json();
      return config;
    } catch (error) {
      logger.error("Error loading config:", error);
      throw error;
    }
  }
});

// 创建一个可加载版本以便在组件中处理加载状态
export const loadableConfigAtom = loadable(configAtom);

// 派生原子
export const versionAtom = atom(
  async (get) => (await get(configAtom)).url.apiVersion,
);

export const appVersionAtom = atom(
  async (get) => (await get(configAtom)).version,
);

export const urlHostAtom = atom(
  async (get) => (await get(configAtom)).url.host,
);

export const urlApiBaseAtom = atom(
  async (get) => (await get(configAtom)).url.apiBase,
);

export const urlWebsiteBaseAtom = atom(
  async (get) => (await get(configAtom)).url.websiteBase,
);

// 派生原子 - 按类别导出
export const grafanaOverviewAtom = atom(
  async (get) => (await get(configAtom)).grafana.overview,
);
export const grafanaNodeAtom = atom(
  async (get) => (await get(configAtom)).grafana.node,
);
export const grafanaJobAtom = atom(
  async (get) => (await get(configAtom)).grafana.job,
);
