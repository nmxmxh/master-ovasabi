"use client";

import styled from "styled-components";

export default function Talent() {
  return (
    <Style.Container>
      <figure className="help-us">
        <h1>Help us understand your business needs</h1>
        <img></img>
      </figure>
      <section className="question">
        <span className="number">1 of 7</span>
        <button className="previous">
          <svg xmlns="http://www.w3.org/2000/svg" width={32} height={33} fill="none">
            <path
              stroke="#232323"
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="m14.333 11.965-5 5m0 0 5 5m-5-5h13.334m8.333 0c0-8.284-6.716-15-15-15-8.284 0-15 6.716-15 15 0 8.284 6.716 15 15 15 8.284 0 15-6.716 15-15Z"
            />
          </svg>
        </button>
        <button className="next">
          <svg xmlns="http://www.w3.org/2000/svg" width={33} height={33} fill="none">
            <path
              stroke="#232323"
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="m18.092 21.965 5-5m0 0-5-5m5 5H9.76m21.667 0c0-8.284-6.716-15-15-15-8.284 0-15 6.716-15 15 0 8.284 6.716 15 15 15 8.284 0 15-6.716 15-15Z"
            />
          </svg>
        </button>
      </section>
      <article className="answers">
        <h2>Select one (or more) of what you currently need or want to improve:</h2>
        <button>A well-structured website or landing page with intuitive user experience</button>
        <button>Engaging social media channels that align with our business goals</button>
        <button>A Google My Business listing to improve local discoverability</button>
        <p>** &nbsp;Your selections help us understand where you’re at, and how we can move you forward.</p>
        <figure className="navigation">
          <h4>Navigation</h4>
          <p>Press the left and right arrow on your keyboard to navigate.</p>
          <div></div>
        </figure>
      </article>
    </Style.Container>
  );
}

const Style = {
  Container: styled.main`
    height: 100dvh;
    width: var(--max-percentage-width);
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    margin-top: 7.5%;

    article.answers {
      width: 22%;
      margin-top: 2.5%;

      figure.navigation {
        margin-top: 32px;
        border: 0.8px solid #252525;
        border-radius: 16px;
        padding: 16px;
        display: flex;
        flex-direction: column;
        align-items: center;

        h4 {
          font-family: "Gordita";
          font-weight: 700;
          font-size: 16px;
          line-height: 150%;
          letter-spacing: -0.11rem;
          text-align: center;
        }

        p {
          max-width: 75%;
          font-family: "Gordita";
          font-weight: 500;
          font-size: 14px;
          line-height: 150%;
          letter-spacing: -0.05rem;
          text-align: center;
          margin-top: 8px;
        }
      }

      p {
        font-family: "Gordita";
        font-weight: 500;
        font-size: 13px;
        line-height: 150%;
        letter-spacing: -0.05rem;
        color: #312420;
        margin-top: 32px;
      }

      h2 {
        font-family: "Gordita";
        font-weight: 700;
        font-size: 18px;
        line-height: 150%;
        letter-spacing: -0.11rem;
      }

      button {
        background: #e5d8c1;
        width: 100%;
        padding: 5% 7.5%;
        padding-right: 12.5%;
        border-radius: 12px;
        font-family: "Gordita";
        font-weight: 500;
        font-size: 15px;
        line-height: 150%;
        letter-spacing: -0.05rem;
        color: #1f1f1f;
        text-align: left;
        margin-top: 16px;
      }
    }

    section.question {
      width: 50%;
      height: 85%;
      background: #f1debb;
      border-radius: 16px;
      position: relative;
      display: flex;
      flex-direction: column;
      align-items: center;
      border: 0.8px solid #252525;
      box-shadow:
        0 5px 10px rgba(0, 0, 0, 0.15),
        0 10px 20px rgba(0, 0, 0, 0.1);

      span.number {
        font-family: "Gordita";
        font-weight: 700;
        font-size: 18px;
        line-height: 150%;
        letter-spacing: -0.13rem;
        text-align: center;
        color: #333336;
        position: absolute;
        top: 5%;
      }

      button {
        position: absolute;
        top: 45%;

        &:first-of-type {
          left: 5%;
        }

        &:last-of-type {
          right: 5%;
        }
      }
    }

    figure.help-us {
      width: 25%;
      height: 75%;
      margin-top: 2.5%;

      img {
        width: 100%;
        height: 60%;
        border: 1px solid red;
        margin-top: 32px;
        border-radius: 12px;
      }

      h1 {
        font-family: "Geist";
        font-weight: 600;
        font-size: 32px;
        line-height: 120%;
        letter-spacing: -0.11rem;
        vertical-align: middle;
        color: #000000;
      }
    }
  `,
};
