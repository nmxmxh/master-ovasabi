import styled from "styled-components";
import { business_dialogue } from "../dialogue/business";
import _ from "lodash";

import RightArrow from "../components/icons/right-arrow";
import LeftArrow from "../components/icons/left-arrow";
import { parseStyledText } from "../lib/utils/parseStyledText";
import { useBusinessLayout } from "../components/shared/layouts/business";

export default function Business() {
  const {
    currentIndex,
    totalQuestions,
    questionRef,
    currentQuestion,
    prev,
    next,
    optionsRef,
    toggleOption,
    currentKey,
    selectedOptions,
    captionRef,
    numberRef,
    nextRef,
    previousRef,
  } = useBusinessLayout();

  return (
    <Style.Container>
      <figure className="help-us">
        <h1>{business_dialogue.main_text}</h1>
        <img></img>
      </figure>
      <section className="question-container" aria-labelledby="question-title" aria-describedby="question-caption">
        <span className="number" ref={numberRef}>
          {currentIndex} of {totalQuestions}
        </span>
        <article className="question">
          <h2 key={currentQuestion.question} ref={questionRef} id="question-title">
            {parseStyledText(currentQuestion.question)}
          </h2>
        </article>
        <figure className="why-this-matters" ref={captionRef} key={currentQuestion.why_this_matters}>
          <h3>{business_dialogue.question_subtitle}</h3>
          <figcaption id="question-caption">{currentQuestion.why_this_matters}</figcaption>
        </figure>
        <button className="previous" onClick={prev} ref={previousRef}>
          <LeftArrow />
        </button>
        <button className="next" onClick={next} ref={nextRef}>
          <RightArrow />
        </button>
      </section>
      <article
        className="answers"
        ref={optionsRef}
        key={`Options for: ${currentQuestion.question}`}
        role="group"
        aria-label={`Options for: ${currentQuestion.question}`}
      >
        <h2>{business_dialogue.options_title}</h2>
        {Object.entries(currentQuestion.options).map(([optionId, label]) => (
          <Style.Option
            key={optionId}
            onClick={() => toggleOption(optionId)}
            id="question-option"
            $isActive={selectedOptions[currentKey]?.includes(optionId)}
          >
            {label}
          </Style.Option>
        ))}
        <p>** &nbsp;{business_dialogue.options_subtitle}</p>
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

    article.question {
      width: 85%;
      position: absolute;
      top: 10%;
      height: 60%;
      padding: 1.5%;

      h2 {
        font-family: "Geist";
        font-weight: 600;
        font-size: 48px;
        line-height: 150%;
        letter-spacing: -0.05rem;
        text-align: center;

        & > div.word {
          border: 1px solid #080808;
          background: #e1e1e1;
          border-radius: 48px;
          padding: 0.5% 3.5%;
          margin: -0.2em;
          transform-origin: bottom center;
        }
      }

      .word,
      .light-light,
      .dark-light,
      .no-split {
        will-change: transform, opacity;
        backface-visibility: hidden;
        transform-style: preserve-3d;
        display: border-box;
      }

      .no-split {
        display: inline;
        border: 1px solid #080808;
        background: #e1e1e1;
        border-radius: 48px;
        padding: 1.5% 2.5%;
        z-index: 1;
      }

      .light-light {
        border: 1px solid #080808;
        background: #cd7755;
        color: #030303;
        border-radius: 48px;
        padding: 1.75% 2.75%;
        margin: -0.2em;
        display: inline;
        z-index: 2;
      }

      .dark-light {
        background: #563123;
        color: #fafafa;
        border-radius: 48px;
        padding: 1.75% 2.75%;
        margin: -0.2em;
        border: 1px solid #080808;
        display: inline;
        z-index: 3;
      }
    }

    figure.why-this-matters {
      width: 85%;
      position: absolute;
      top: 75%;
      border: 1px solid #ead7b6;
      background: #f6edde;
      border-radius: 12px;
      height: 20%;
      padding: 2.5%;

      h3 {
        font-family: "gordita";
        font-weight: 700;
        font-size: 18px;
        line-height: 150%;
        letter-spacing: -0.11rem;
        margin-bottom: 1%;
      }

      figcaption {
        font-family: "Gordita";
        font-weight: 500;
        font-size: 15px;
        line-height: 150%;
        letter-spacing: -0.05rem;
      }
    }

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
        letter-spacing: -0.07rem;
      }
    }

    section.question-container {
      width: 50%;
      height: 85%;
      background: #f1debb;
      border-radius: 16px;
      position: relative;
      display: flex;
      flex-direction: column;
      align-items: center;
      border: 0.8px solid #252525;

      span.number {
        font-family: "Gordita";
        font-weight: 700;
        font-size: 18px;
        line-height: 150%;
        letter-spacing: -0.1rem;
        text-align: center;
        color: #333336;
        position: absolute;
        top: 5%;
      }

      button {
        position: absolute;
        top: 45%;
        height: 4.5%;

        &:first-of-type {
          left: 2.5%;
        }

        &:last-of-type {
          right: 2.5%;
        }

        svg {
          height: 100%;
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
        font-family: "geist";
        font-weight: 600;
        font-size: 32px;
        line-height: 120%;
        letter-spacing: -0.11rem;
        vertical-align: middle;
        color: #000000;
      }
    }
  `,
  Option: styled.button<{ $isActive?: boolean }>`
    background: ${({ $isActive }) => ($isActive ? "#CD7755" : "#e5d8c1")};
    border: ${({ $isActive }) => ($isActive ? "0.8px solid #252525" : "0.8px solid transparent")};
    color: ${({ $isActive }) => ($isActive ? "#FAFAFA" : "#1f1f1f")};
    width: 100%;
    padding: 5% 7.5%;
    padding-right: 12.5%;
    border-radius: 12px;
    font-family: "Gordita";
    font-weight: 500;
    font-size: 15px;
    line-height: 150%;
    letter-spacing: -0.05rem;
    text-align: left;
    margin-top: 16px;
    transition:
      background 0.25s ease-in,
      border 0.25s ease-out,
      color 0.25s linear;

    &:hover {
      border: 0.8px solid #252525;
    }
  `,
};
