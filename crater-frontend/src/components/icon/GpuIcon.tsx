import { SVGProps, Ref, forwardRef } from "react";
const SvgComponent = (
  props: SVGProps<SVGSVGElement>,
  ref: Ref<SVGSVGElement>,
) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width={24}
    height={24}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
    strokeLinecap="round"
    strokeLinejoin="round"
    strokeWidth={2}
    ref={ref}
    {...props}
  >
    <title>{"Layer 1"}</title>
    <path
      stroke="null"
      d="M22.094 14.906V6.187a1.938 2.019 90 0 0-2.019-1.937H3.925a1.938 2.019 90 0 0-2.019 1.937v11.625a1.938 2.019 90 0 0 2.019 1.938h13.122l5.047-4.844zM1.875 2.844v18.312"
    />
    <circle cx={12} cy={12} r={5} />
    <path d="M12 17a2.5 2.5 0 0 0 0-5 2.5 2.5 0 0 1 0-5" />
    <path d="M7 12a2.5 2.5 0 0 0 5 0 2.5 2.5 0 0 1 5 0" />
  </svg>
);
const ForwardRef = forwardRef(SvgComponent);
export default ForwardRef;
