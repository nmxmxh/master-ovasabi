import type { ReactNode } from 'react';
import { useEffect } from 'react';
import styled from 'styled-components';
import Header from '../header';
import Footer from '../footer';

function ScrollToTop() {
  useEffect(() => {
    window.scrollTo(0, 0);
  }, []);
  return null;
}

export default function PublicLayout({ children }: { children: ReactNode }) {
  return (
    <Style.Container>
      <ScrollToTop />
      <Header />
      {children}
      <Footer />
    </Style.Container>
  );
}

const Style = {
  Container: styled.div`
    height: 110dvh;
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    z-index: 1;
  `
};
