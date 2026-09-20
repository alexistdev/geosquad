import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';

import { AuthService } from '../services/auth.service';

/**
 * Kebalikan authGuard: menahan user yang sudah login agar tidak membuka
 * halaman login atau registrasi lagi, dan melemparnya ke dashboard.
 */
export const guestGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);

  const user = auth.user();
  return user ? router.createUrlTree([auth.homeUrlFor(user.role)]) : true;
};
