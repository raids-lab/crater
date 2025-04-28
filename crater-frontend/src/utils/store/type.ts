export interface AppConfig {
  version: string;
  url: {
    host: string;
    apiBase: string;
    apiVersion: string;
    websiteBase: string;
  };
  grafana: {
    overview: {
      main: string;
      schedule: string;
      network: string;
    };
    node: {
      basic: string;
      nvidia: string;
    };
    job: {
      basic: string;
      nvidia: string;
      pod: string;
    };
    user: {
      nvidia: string;
    };
  };
  featureFlags: {
    enableRegister: boolean;
    setDocsAsHome: boolean;
    alertLowCPURequest: boolean;
    enableMockServiceWorker: boolean;
  };
}
