"use client";

import styled from "styled-components";

import { Link } from "react-router-dom";
import Logo from "../icons/logo";

export default function Header() {
  return (
    <Style.Container>
      <Link to="/">
        <Logo />
      </Link>
      <figure>
        <label>accessibility settings</label>
        <button>
          <svg id="menu-button" xmlns="http://www.w3.org/2000/svg" width={25} height={12} fill="none">
            <rect width={25} height={4} fill="#444448" rx={2} />
            <rect width={25} height={4} y={8} fill="#444448" rx={2} />
          </svg>
        </button>
      </figure>
    </Style.Container>
  );
}

export const Style = {
  Container: styled.header`
    height: 10dvh;
    position: absolute;
    top: 0;
    width: var(--max-percentage-width);
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin: auto;
    overflow: hidden;

    a {
      height: 50%;

      svg {
        height: 100%;
      }
    }

    figure {
      label {
        font-family: "Gordita";
        font-weight: 500;
        font-size: 16px;
        line-height: 150%;
        letter-spacing: -0.05rem;
        text-align: center;
        margin-right: 12px;
        color: #444448;
        opacity: 0;
      }
    }
  `,
};
