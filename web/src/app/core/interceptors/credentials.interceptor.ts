import { HttpInterceptorFn } from '@angular/common/http';

/**
 * Menyertakan cookie pada setiap request ke API.
 *
 * Tanpa `withCredentials`, browser tidak mengirim cookie SID dan seluruh
 * request akan dibalas 401 meski user baru saja login.
 */
export const credentialsInterceptor: HttpInterceptorFn = (req, next) =>
  next(req.clone({ withCredentials: true }));
