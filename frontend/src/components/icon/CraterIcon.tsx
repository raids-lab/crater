import { SVGProps, Ref, forwardRef } from "react";
const SvgComponent = (
  props: SVGProps<SVGSVGElement>,
  ref: Ref<SVGSVGElement>,
) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    xmlSpace="preserve"
    viewBox="0 0 180 180"
    ref={ref}
    {...props}
  >
    <path
      d="M163.9 90v28.2c0 8.9-4.8 17.2-12.5 21.7L126.9 154l-24.3 14c-7.8 4.5-17.4 4.5-25.1-.1l-49-28.7c-7.7-4.5-12.4-12.7-12.4-21.6V61.8c0-8.9 4.8-17.2 12.5-21.7l48.9-28.2c7.7-4.5 17.3-4.5 25 0L126.9 26v42.6l36.9-21.3.1 42.7z"
      style={{
        fill: "currentColor",
      }}
    />
    <path
      d="M126.9 26v76.6c0 5.4-2.9 10.3-7.5 13l-21.9 12.7c-4.6 2.7-10.4 2.7-15 0l-24.4-14.1c-3.1-1.8-5-5.1-5-8.7V77.3c0-5.4 2.9-10.3 7.5-13l29.4-17L126.9 26z"
      style={{
        fill: "#de553e",
      }}
    />
    <path
      d="M126.9 68.7v36.9c0 3.6-1.9 6.9-5 8.7l-24.5 14.1c-2.3 1.3-4.9 2-7.5 2V90l37-21.3z"
      style={{
        fill: "#faac3e",
      }}
    />
  </svg>
);
const ForwardRef = forwardRef(SvgComponent);
export default ForwardRef;
