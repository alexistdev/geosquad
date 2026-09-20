import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { map } from 'rxjs';

import { AuthService } from '../services/auth.service';

/**
 * Menjaga halaman yang butuh login.
 *
 * Kalau profil hasil login sebelumnya masih ada, halaman dibuka seketika dan
 * kebenarannya diverifikasi server pada request pertama. Kalau tidak ada --
 * misalnya tab baru atau setelah refresh di perangkat yang localStorage-nya
 * dibersihkan -- barulah bertanya ke /auth/me, karena cookie sesi bisa saja
 * masih hidup meski frontend tidak mengingat apa-apa.
 */
export const authGuard: CanActivateFn = (_route, state) => {
  const auth = inject(AuthService);
  const router = inject(Router);

  if (auth.isLoggedIn()) {
    return true;
  }

  return auth.refresh().pipe(
    map((user) =>
      user ? true : router.createUrlTree(['/login'], { queryParams: { redirect: state.url } }),
    ),
  );
};
