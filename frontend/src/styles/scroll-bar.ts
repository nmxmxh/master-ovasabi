import { css } from "styled-components";

export function scrollBarLook() {
  return css`
    &::-webkit-scrollbar {
      width: 5px;
      background: transparent;
      z-index: 1;
    }

    &::-webkit-scrollbar-thumb {
      background: black;
      box-shadow: -2px 4px 10px 0px rgba(0, 0, 0, 0.2);
      z-index: 2;
      width: 5px;
    }

    /* Firefox specific styles */
    @supports (-moz-appearance: none) {
      scrollbar-color: black transparent;
      scrollbar-width: thin;

      & {
        /* Disable default Firefox scrollbar */
        -moz-appearance: none;
        appearance: none;
      }

      &::-moz-range-track {
        /* Border radius for the scrollbar track in Firefox */
        border-radius: 4px;
      }
    }
  `;
}
