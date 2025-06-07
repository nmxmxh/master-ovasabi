import type { IconProps } from "../../types/icons";

const LeftArrow = (props: IconProps) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    preserveAspectRatio="xMidYMid meet"
    viewBox="0 0 33 33"
    fill="none"
    {...props}
  >
    <path
      stroke="#232323"
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="m14.333 11.674-5 5m0 0 5 5m-5-5h13.334m8.333 0c0-8.284-6.716-15-15-15-8.284 0-15 6.716-15 15 0 8.284 6.716 15 15 15 8.284 0 15-6.716 15-15Z"
    />
  </svg>
);
export default LeftArrow;
