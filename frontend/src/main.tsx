import {StrictMode} from 'react';
import {createRoot} from 'react-dom/client';
import {registerSW} from 'virtual:pwa-register';
import App from './App.tsx';
import {AdminApp} from './screens/AdminApp.tsx';
import './index.css';

registerSW({immediate: true});

const isAdminRoute = window.location.pathname.startsWith('/admin');

createRoot(document.getElementById('root')!).render(
    <StrictMode>
        {isAdminRoute ? <AdminApp/> : <App/>}
    </StrictMode>
);
