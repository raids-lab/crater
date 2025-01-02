export interface MetadataFormType {
  version: string;
  type: string;
}

export const MetadataFormAccount: MetadataFormType = {
  version: "20241208",
  type: "account",
};

export const MetadataFormJupyter: MetadataFormType = {
  version: "20241217",
  type: "jupyter",
};

export const MetadataFormDockerfile: MetadataFormType = {
  version: "20241208",
  type: "dockerfile",
};

export const MetadataFormPipApt: MetadataFormType = {
  version: "20250102",
  type: "pipapt",
};
