import nivoDarkTheme from "@/components/chart/darkTheme";
import nivoLightTheme from "@/components/chart/defaultTheme";
import { useTheme } from "@/utils/theme";

export const useNivoTheme = () => {
  const { theme } = useTheme();
  return {
    nivoTheme:
      theme === "light"
        ? nivoLightTheme
        : { ...nivoDarkTheme, background: "#10172a" },
    theme: theme,
  };
};
