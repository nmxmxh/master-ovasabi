"use client";

import { Link } from "react-router-dom";
import styled from "styled-components";

export default function Footer() {
  return (
    <Style.Container>
      <button>Ovasabi Studios.</button>
      <nav>
        <Link to="about-us">about us</Link>
        <Link to="shop">shop</Link>
        <Link to="projects">projects</Link>
      </nav>
    </Style.Container>
  );
}

export const Style = {
  Container: styled.footer`
    position: absolute;
    bottom: 0;
    height: 10dvh;
    position: absolute;
    width: var(--max-percentage-width);
    margin: auto;
    display: flex;
    align-items: center;
    justify-content: space-between;

    button {
      font-family: "Geist";
      font-weight: 600;
      font-size: 22px;
      letter-spacing: -0.05rem;
      vertical-align: middle;
      color: #f2f1f1;
      color: #241f1f;
    }

    nav {
      a {
        margin-left: 54px;
        font-family: "Gordita";
        font-weight: 700;
        font-size: 16px;
        line-height: 150%;
        letter-spacing: -0.05rem;
        text-align: center;
      }
    }
  `,
};
