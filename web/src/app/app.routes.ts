import { Routes } from '@angular/router';

import { authGuard } from './core/guards/auth.guard';
import { guestGuard } from './core/guards/guest.guard';
import { roleGuard } from './core/guards/role.guard';
import { Shell } from './shared/layout/shell';

/**
 * Semua halaman di-lazy load. Bundle awal cukup memuat login, sehingga
 * pengunjung yang belum masuk tidak perlu mengunduh layar admin.
 */
export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'login' },

  {
    path: 'login',
    canActivate: [guestGuard],
    loadComponent: () => import('./pages/login/login').then((m) => m.Login),
  },
  {
    path: 'register',
    canActivate: [guestGuard],
    loadComponent: () => import('./pages/register/register').then((m) => m.Register),
  },

  {
    // Semua rute di dalam sini berbagi satu kerangka halaman, dan authGuard
    // cukup dipasang sekali di induknya.
    path: '',
    component: Shell,
    canActivate: [authGuard],
    children: [
      {
        path: 'user/dashboard',
        loadComponent: () => import('./pages/dashboard/dashboard').then((m) => m.Dashboard),
      },
      {
        path: 'admin/dashboard',
        canActivate: [roleGuard],
        data: { roles: ['ADMIN'] },
        loadComponent: () => import('./pages/dashboard/dashboard').then((m) => m.Dashboard),
      },
      {
        path: 'admin/users',
        canActivate: [roleGuard],
        data: { roles: ['ADMIN'] },
        loadComponent: () => import('./pages/admin-users/admin-users').then((m) => m.AdminUsers),
      },
      {
        path: 'runs/:id',
        loadComponent: () => import('./pages/run-detail/run-detail').then((m) => m.RunDetail),
      },
      {
        path: '**',
        loadComponent: () => import('./pages/not-found/not-found').then((m) => m.NotFound),
      },
    ],
  },

  {
    path: '**',
    loadComponent: () => import('./pages/not-found/not-found').then((m) => m.NotFound),
  },
];
