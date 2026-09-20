import {
  ApplicationConfig,
  provideBrowserGlobalErrorListeners,
  provideZoneChangeDetection,
} from '@angular/core';
import { provideRouter, withInMemoryScrolling } from '@angular/router';
import { provideHttpClient, withFetch, withInterceptors } from '@angular/common/http';

import { routes } from './app.routes';
import { credentialsInterceptor } from './core/interceptors/credentials.interceptor';
import { errorInterceptor } from './core/interceptors/error.interceptor';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideZoneChangeDetection({ eventCoalescing: true }),
    provideRouter(
      routes,
      // Tanpa ini, pindah halaman mempertahankan posisi scroll halaman
      // sebelumnya, sehingga halaman baru terbuka di tengah-tengah.
      withInMemoryScrolling({ scrollPositionRestoration: 'top' }),
    ),
    provideHttpClient(
      withFetch(),
      // Urutannya penting: credentials memasang cookie pada request, error
      // menangkap balasannya. Interceptor pertama membungkus yang kedua.
      withInterceptors([credentialsInterceptor, errorInterceptor]),
    ),
  ],
};
