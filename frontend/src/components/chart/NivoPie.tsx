import useNivoTheme from "@/hooks/useNivoTheme";
import { MayHaveLabel, PieSvgProps, ResponsivePie } from "@nivo/pie";

const NivoPie = <RawDatum extends MayHaveLabel>(
  props: Omit<PieSvgProps<RawDatum>, "width" | "height">,
) => {
  const { nivoTheme, theme } = useNivoTheme();

  return (
    <ResponsivePie
      innerRadius={0.5}
      padAngle={0.7}
      cornerRadius={3}
      activeOuterRadiusOffset={8}
      borderColor={{
        from: "color",
        modifiers: [["darker", 0.6]],
      }}
      arcLinkLabelsSkipAngle={10}
      arcLinkLabel={(d) => `${d.label || d.id}`}
      arcLabelsSkipAngle={10}
      colors={{ scheme: theme === "dark" ? "category10" : "paired" }}
      theme={nivoTheme}
      {...props}
    />
  );
};

export default NivoPie;
