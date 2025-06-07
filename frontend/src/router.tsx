import { createBrowserRouter } from 'react-router-dom';
import { Home } from './pages/home';
import Business from './pages/business';
import PublicLayout from './components/shared/layouts/public';
import Pioneer from './pages/pioneer';
import Talent from './pages/talent';
import Hustler from './pages/hustler';
import SearchDemoPage from './pages/search-demo';

export const router = createBrowserRouter([
  {
    path: '/pioneer',
    element: (
      <PublicLayout>
        <Pioneer />
      </PublicLayout>
    )
  },
  {
    path: '/talent',
    element: (
      <PublicLayout>
        <Talent />
      </PublicLayout>
    )
  },
  {
    path: '/hustler',
    element: (
      <PublicLayout>
        <Hustler />
      </PublicLayout>
    )
  },
  {
    path: '/business',
    element: (
      <PublicLayout>
        <Business />
      </PublicLayout>
    )
  },
  {
    path: '/search-demo',
    element: <SearchDemoPage />
  },
  {
    path: '/',
    element: (
      <PublicLayout>
        <Home />
      </PublicLayout>
    )
  }
]);
