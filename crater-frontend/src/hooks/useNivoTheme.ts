import nivoDarkTheme from "@/components/chart/darkTheme";
import nivoLightTheme from "@/components/chart/defaultTheme";
import { Theme } from "@nivo/core";
import { useTheme } from "@/utils/theme";

/**
 * `useNivoTheme` is a custom hook that returns the Nivo theme based on the current application theme.
 */
const useNivoTheme = (): { nivoTheme: Theme; theme: string } => {
  const { theme } = useTheme();
  return {
    nivoTheme:
      theme === "light"
        ? nivoLightTheme
        : { ...nivoDarkTheme, background: "#10172a" },
    theme: theme,
  };
};

export default useNivoTheme;
