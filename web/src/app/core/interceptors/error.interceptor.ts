import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError, throwError } from 'rxjs';

import { AuthService } from '../services/auth.service';
import { BaseResponse } from '../models/response';

/**
 * Menerjemahkan error HTTP jadi pesan yang bisa ditampilkan, dan menendang
 * user keluar saat sesinya tidak lagi diterima server.
 */
export const errorInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);

  return next(req).pipe(
    catchError((error: HttpErrorResponse) => {
      // 401 di luar proses login berarti sesinya mati: kedaluwarsa, dicabut
      // admin, atau user menekan logout di tab lain.
      const isLoginAttempt = req.url.includes('/auth/login');
      if (error.status === 401 && !isLoginAttempt) {
        auth.forceLogout();
      }

      return throwError(() => new Error(messageFrom(error)));
    }),
  );
};

function messageFrom(error: HttpErrorResponse): string {
  // API selalu membalas {status, messages, payload}, termasuk saat gagal.
  const body = error.error as BaseResponse<unknown> | null;
  if (body?.messages?.length) {
    return body.messages.join(' ');
  }

  if (error.status === 0) {
    return 'Tidak bisa menghubungi server. Pastikan API sedang berjalan.';
  }
  return `Terjadi kesalahan (HTTP ${error.status}).`;
}
