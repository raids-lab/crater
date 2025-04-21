export interface MetadataFormType {
  version: string;
  type: string;
}

export const MetadataFormAccount: MetadataFormType = {
  version: "20241208",
  type: "account",
};

export const MetadataFormJupyter: MetadataFormType = {
  version: "20250420",
  type: "jupyter",
};

export const MetadataFormCustom: MetadataFormType = {
  version: "20250317",
  type: "custom",
};

export const MetadataFormCustomEmias: MetadataFormType = {
  version: "20250420",
  type: "custom-emias",
};

export const MetadataFormTensorflow: MetadataFormType = {
  version: "20240528",
  type: "tensorflow",
};

export const MetadataFormPytorch: MetadataFormType = {
  version: "20240528",
  type: "pytorch",
};

export const MetadataFormDockerfile: MetadataFormType = {
  version: "20241208",
  type: "dockerfile",
};

export const MetadataFormPipApt: MetadataFormType = {
  version: "20250102",
  type: "pipapt",
};

export const MetadataFormJupyterEmias: MetadataFormType = {
  version: "20240528",
  type: "jupyter-emias",
};
