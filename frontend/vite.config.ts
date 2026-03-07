import {defineConfig} from 'vite';
import {VitePWA} from 'vite-plugin-pwa';
import react from '@vitejs/plugin-react';

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [
        react(),
        VitePWA({
            strategies: 'injectManifest',
            srcDir: 'src',
            filename: 'sw.ts',
            registerType: 'autoUpdate',
            includeAssets: ['icon-*.png'],
            manifest: {
                name: 'Personage - Personal Assistant',
                short_name: 'Personage',
                description: 'Персональный ассистент для управления задачами и расписанием',
                start_url: '/',
                display: 'standalone',
                background_color: '#F7F8FA',
                theme_color: '#5C6BFF',
                orientation: 'portrait-primary',
                categories: ['productivity', 'utilities'],
                icons: [
                    {src: 'icon-72x72.png', sizes: '72x72', type: 'image/png', purpose: 'any maskable'},
                    {src: 'icon-96x96.png', sizes: '96x96', type: 'image/png', purpose: 'any maskable'},
                    {src: 'icon-128x128.png', sizes: '128x128', type: 'image/png', purpose: 'any maskable'},
                    {src: 'icon-144x144.png', sizes: '144x144', type: 'image/png', purpose: 'any maskable'},
                    {src: 'icon-152x152.png', sizes: '152x152', type: 'image/png', purpose: 'any maskable'},
                    {src: 'icon-192x192.png', sizes: '192x192', type: 'image/png', purpose: 'any maskable'},
                    {src: 'icon-384x384.png', sizes: '384x384', type: 'image/png', purpose: 'any maskable'},
                    {src: 'icon-512x512.png', sizes: '512x512', type: 'image/png', purpose: 'any maskable'},
                ],
                shortcuts: [
                    {
                        name: 'Задачи',
                        short_name: 'Задачи',
                        description: 'Открыть список задач',
                        url: '/?screen=tasks',
                        icons: [{src: 'icon-96x96.png', sizes: '96x96'}],
                    },
                    {
                        name: 'Расписание',
                        short_name: 'Расписание',
                        description: 'Открыть расписание',
                        url: '/?screen=schedule',
                        icons: [{src: 'icon-96x96.png', sizes: '96x96'}],
                    },
                ],
            },
            injectManifest: {
                globPatterns: ['**/*.{js,css,html,ico,png,svg,woff,woff2}'],
            },
            devOptions: {
                enabled: true,
                type: 'module',
            },
        }),
    ],
    optimizeDeps: {
        exclude: ['lucide-react'],
    },
});
