// exportToJson convert T to JSON and save it to file and download it.
export const exportToJson = (data: unknown, filename = "data.json") => {
  const json = JSON.stringify(data, null, 2);
  const blob = new Blob([json], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
};

// usage: importFromJson<FormData>(file)
export const importFromJson = async <T>(file?: File): Promise<T> => {
  if (!file) {
    throw new Error("No file provided");
  }
  const text = await file.text();
  try {
    return JSON.parse(text) as T;
  } catch (error) {
    throw new Error("Invalid JSON file");
  }
};
