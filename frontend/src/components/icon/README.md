# Project Icon Components

## Resource

- Radix Icon: https://www.radix-ui.com/icons
- IconPark: https://iconpark.oceanengine.com/

## Convert to component

If _IconPark_ package is introduced, the Vite bundling time will be significantly extended. Furthermore, IconPark does not provide good support for Tailwind CSS.

Therefore, it is recommended to manually import the components and convert the SVG files into React components with [React SVGR](https://react-svgr.com/playground/?ref=true&replaceAttrValues=%23fff%3DcurrentColor&typescript=true).
