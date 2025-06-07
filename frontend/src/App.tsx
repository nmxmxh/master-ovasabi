import { RouterProvider } from 'react-router-dom';
import { router } from './router';
import { ComingSoon } from './components/shared/coming-soon';

function App() {
  return (
    <>
      {import.meta.env.PROD && <ComingSoon />}
      {!import.meta.env.PROD && <RouterProvider router={router} />}
    </>
  );
}

export default App;
