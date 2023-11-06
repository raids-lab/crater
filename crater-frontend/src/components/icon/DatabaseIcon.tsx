import { SVGProps, Ref, forwardRef } from "react";
const SvgComponent = (
  props: SVGProps<SVGSVGElement>,
  ref: Ref<SVGSVGElement>,
) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width={15}
    height={15}
    fill="none"
    viewBox="0 0 48 48"
    ref={ref}
    {...props}
  >
    <path
      stroke="currentColor"
      strokeLinecap="square"
      strokeWidth={3}
      d="M44 11v27c0 3.314-8.954 6-20 6S4 41.314 4 38V11"
    />
    <path
      stroke="currentColor"
      strokeLinecap="square"
      strokeWidth={3}
      d="M44 29c0 3.314-8.954 6-20 6S4 32.314 4 29M44 20c0 3.314-8.954 6-20 6S4 23.314 4 20"
    />
    <ellipse
      cx={24}
      cy={10}
      stroke="currentColor"
      strokeLinecap="square"
      strokeWidth={3}
      rx={20}
      ry={6}
    />
  </svg>
);
const ForwardRef = forwardRef(SvgComponent);
export default ForwardRef;
