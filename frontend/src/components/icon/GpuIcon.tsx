import { SVGProps, Ref, forwardRef } from "react";
const SvgComponent = (
  props: SVGProps<SVGSVGElement>,
  ref: Ref<SVGSVGElement>,
) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width={24}
    height={24}
    aria-hidden="true"
    viewBox="0 0 16 16"
    ref={ref}
    {...props}
  >
    <g fill="none">
      <g clipPath="url(#a)">
        <path
          fill="currentColor"
          fillRule="evenodd"
          d="M0 1.25A.75.75 0 0 1 .75.5h.5A2.25 2.25 0 0 1 3.5 2.75V3H13a3 3 0 0 1 3 3v4a3 3 0 0 1-3 3H3.5v1.25a.75.75 0 0 1-1.5 0V2.75A.75.75 0 0 0 1.25 2h-.5A.75.75 0 0 1 0 1.25M13 4.5H3.5v7H13a1.5 1.5 0 0 0 1.5-1.5V6A1.5 1.5 0 0 0 13 4.5M8.5 8a1 1 0 1 1-2 0 1 1 0 0 1 2 0M10 8a2.5 2.5 0 1 1-5 0 2.5 2.5 0 0 1 5 0"
          clipRule="evenodd"
        />
      </g>
      <defs>
        <clipPath id="a">
          <path fill="currentColor" d="M0 0h16v16H0z" />
        </clipPath>
      </defs>
    </g>
  </svg>
);
const ForwardRef = forwardRef(SvgComponent);
export default ForwardRef;
