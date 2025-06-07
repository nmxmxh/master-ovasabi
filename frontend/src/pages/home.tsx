import { Link } from "react-router-dom";
import styled from "styled-components";

export function Home() {
  return (
    <Style.Container>
      <h1>
        <span>
          Choose <br />
          Your Path
        </span>{" "}
        <svg xmlns="http://www.w3.org/2000/svg" width={41} height={9} fill="none">
          <rect width={40} height={8} x={0.566} y={0.5} fill="#1A1A1A" rx={2} />
        </svg>
        <span>
          Shape Your <br />
          Future{" "}
          <svg xmlns="http://www.w3.org/2000/svg" width={73} height={47} fill="none">
            <rect width={71} height={45.857} x={1.434} y={0.5} fill="#EBAD41" stroke="#000" rx={22.929} />
            <path
              fill="#343434"
              d="M54.39 15.213a1.5 1.5 0 0 0-1.1-1.813l-13.114-3.206a1.5 1.5 0 0 0-.712 2.914l11.656 2.85-2.85 11.656a1.5 1.5 0 0 0 2.915.713l3.206-13.114ZM20.935 34.285l.778 1.282 32-19.428-.778-1.282-.779-1.283-32 19.429.779 1.282Z"
            />
          </svg>
        </span>
      </h1>
      <h2>
        At Ovasabi, we believe everyone carries a unique spark. Some build, some sell, some lead, some create. Pick the
        card that calls to you: Pioneer. Business. Talent. Salesman.
      </h2>
      <div className="home-spacebar">
        <svg xmlns="http://www.w3.org/2000/svg" width={32} height={6} fill="none">
          <rect width={32} height={6} fill="#453D3D" rx={2} />
        </svg>
        <p>or press Spacebar and let us take you on a journey.</p>
      </div>
      <nav className="home-choices">
        <Link to="/business">
          <h3>
            <span>FOR</span>
            <br />
            Businesses
          </h3>
          <figure></figure>
        </Link>
        <Link to="/talent">
          <h3>
            <span>FOR</span>
            <br />
            Talent
          </h3>
          <figure></figure>
        </Link>
        <Link to="/pioneer">
          <h3>
            <span>FOR</span>
            <br />
            Pioneers
          </h3>
          <figure></figure>
        </Link>
        <Link to="/hustler">
          <h3>
            <span>FOR</span>
            <br />
            Hustlers
          </h3>
          <figure></figure>
        </Link>
      </nav>
    </Style.Container>
  );
}

const Style = {
  Container: styled.main`
    height: 100%;
    width: var(--max-percentage-width);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-direction: column;

    .home-choices {
      height: 42.5%;
      width: 80%;
      margin-bottom: 2.5%;
      display: flex;
      align-items: center;
      justify-content: space-between;

      a {
        height: 100%;
        border: 0.8px solid #030303;
        width: 23.5%;
        border-radius: 12px;
        padding: 8px;
        padding-top: 16px;
        z-index: 2;

        &:hover {
          z-index: 3;
        }

        &:first-of-type {
          background: #f9ebdc;
        }

        &:nth-of-type(2) {
          background: #dcf1f9;
        }

        &:nth-of-type(3) {
          background: #f9ebdc;
        }

        &:last-of-type {
          background: #eeeeeb;
        }

        figure {
          height: 75%;
        }

        h3 {
          color: #000000;
          font-family: "geist";
          font-weight: 600;
          font-size: 32px;
          line-height: 55%;
          letter-spacing: -0.11rem;
          vertical-align: middle;
          display: flex;
          flex-direction: column;
          align-items: flex-start;
          margin-top: 2.5%;

          span {
            font-family: "geist";
            font-weight: 500;
            font-size: 16px;
            letter-spacing: -0.05rem;
            line-height: 0%;
            text-transform: uppercase;
          }
        }
      }
    }

    div.home-spacebar {
      display: flex;
      flex-direction: column;
      align-items: center;
      margin: 32px 0;

      svg {
        margin-bottom: 16px;
      }

      p {
        font-family: "gordita";
        font-weight: 700;
        font-size: 16px;
        line-height: 150%;
        letter-spacing: -0.05rem;
        text-align: center;
        color: #444448;
      }
    }

    h1 {
      font-family: "geist";
      font-weight: 700;
      font-size: 40px;
      line-height: 120%;
      letter-spacing: -0.1rem;
      vertical-align: middle;
      display: flex;
      align-items: center;
      margin-bottom: 30px;

      .home-path-stroke {
        color: #ebad41;
        -webkit-text-stroke-width: 2px;
        -webkit-text-stroke-color: black;
      }

      & > svg {
        margin: 0 30px;
      }
    }

    h2 {
      font-family: "gordita";
      font-weight: 500;
      font-size: 18px;
      line-height: 150%;
      letter-spacing: -0.05rem;
      text-align: center;
      max-width: 60%;
      color: #444448;
    }
  `,
};
