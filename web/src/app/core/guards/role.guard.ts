import { inject } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivateFn, Router } from '@angular/router';

import { AuthService } from '../services/auth.service';
import { Role } from '../models/user';

/**
 * Membatasi halaman ke peran tertentu. Dipasang setelah authGuard, yang sudah
 * memastikan profil terisi.
 *
 * Penolakan mengarahkan ke dashboard peran yang bersangkutan, bukan ke login:
 * user ini sah, cuma salah alamat.
 */
export const roleGuard: CanActivateFn = (route: ActivatedRouteSnapshot) => {
  const auth = inject(AuthService);
  const router = inject(Router);

  const allowed = (route.data['roles'] as Role[] | undefined) ?? [];
  const user = auth.user();

  if (!user) {
    return router.createUrlTree(['/login']);
  }
  if (allowed.length === 0 || allowed.includes(user.role)) {
    return true;
  }
  return router.createUrlTree([auth.homeUrlFor(user.role)]);
};
