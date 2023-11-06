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
      d="M6 44h36M17 20h14"
    />
    <path stroke="currentColor" strokeWidth={3} d="M19 20h10l3 24H16l3-24Z" />
    <path
      stroke="currentColor"
      strokeLinecap="square"
      strokeWidth={3}
      d="m19 9 2 11h6l2-11"
    />
    <path
      stroke="currentColor"
      strokeLinecap="square"
      strokeWidth={3}
      d="m32 12-3-3-5-5-5 5-3 3M37 7l3-3M11 7 8 4M37 19l3 3M11 19l-3 3M38 13h4M10 13H6M18 28h12M17 36h14"
    />
    <path stroke="currentColor" strokeWidth={3} d="m29 20 3 24M19 20l-3 24" />
  </svg>
);
const ForwardRef = forwardRef(SvgComponent);
export default ForwardRef;
