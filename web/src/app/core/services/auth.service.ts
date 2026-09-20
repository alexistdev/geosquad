import { Injectable, computed, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { Observable, tap, of, catchError, map } from 'rxjs';

import { environment } from '../../../environments/environment';
import { BaseResponse } from '../models/response';
import {
  ChangePasswordRequest,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  Role,
  User,
} from '../models/user';

/**
 * Sumber kebenaran status login.
 *
 * Sesi dipegang cookie `SID` yang HttpOnly, artinya JavaScript tidak bisa
 * membacanya. Konsekuensinya frontend tidak bisa "melihat" apakah dirinya
 * login hanya dari cookie -- satu-satunya cara memastikan adalah bertanya ke
 * server lewat /auth/me. Itu justru bagus: XSS tidak bisa mencuri sesi.
 *
 * Profil user disalin ke localStorage supaya guard bisa memutuskan seketika
 * saat halaman dibuka, tanpa layar kosong menunggu jaringan. Salinan itu hanya
 * petunjuk, bukan otoritas: server tetap memeriksa tiap request, dan salinan
 * yang bohong paling jauh menampilkan menu yang API-nya akan menolak.
 */
@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);
  private readonly base = `${environment.apiUrl}/auth`;

  private static readonly STORAGE_KEY = 'geosquad.user';

  private readonly currentUser = signal<User | null>(readStoredUser());

  readonly user = this.currentUser.asReadonly();
  readonly isLoggedIn = computed(() => this.currentUser() !== null);
  readonly isAdmin = computed(() => this.currentUser()?.role === 'ADMIN');

  login(request: LoginRequest): Observable<LoginResponse> {
    return this.http
      .post<BaseResponse<LoginResponse>>(`${this.base}/login`, request)
      .pipe(
        map((res) => res.payload!),
        tap((payload) => this.setUser(payload.user)),
      );
  }

  register(request: RegisterRequest): Observable<User> {
    return this.http
      .post<BaseResponse<User>>(`${this.base}/register`, request)
      .pipe(map((res) => res.payload!));
  }

  /** Memastikan sesi masih hidup di server dan menyegarkan profil lokal. */
  refresh(): Observable<User | null> {
    return this.http.get<BaseResponse<User>>(`${this.base}/me`).pipe(
      map((res) => res.payload ?? null),
      tap((user) => (user ? this.setUser(user) : this.clearUser())),
      catchError(() => {
        this.clearUser();
        return of(null);
      }),
    );
  }

  changePassword(request: ChangePasswordRequest): Observable<void> {
    return this.http
      .post<BaseResponse<void>>(`${this.base}/change-password`, request)
      .pipe(
        // Server mencabut semua sesi setelah password berganti, termasuk sesi
        // ini. Status lokal harus ikut dibersihkan supaya UI tidak mengira
        // dirinya masih login.
        tap(() => this.clearUser()),
        map(() => void 0),
      );
  }

  logout(): Observable<void> {
    return this.http.post<BaseResponse<void>>(`${this.base}/logout`, {}).pipe(
      catchError(() => of(null)),
      tap(() => this.clearUser()),
      map(() => void 0),
    );
  }

  logoutAll(): Observable<void> {
    return this.http.post<BaseResponse<void>>(`${this.base}/logout-all`, {}).pipe(
      catchError(() => of(null)),
      tap(() => this.clearUser()),
      map(() => void 0),
    );
  }

  /** Dipanggil interceptor saat server membalas 401 di tengah sesi. */
  forceLogout(): void {
    this.clearUser();
    this.router.navigate(['/login'], { queryParams: { expired: '1' } });
  }

  homeUrlFor(role: Role | undefined): string {
    return role === 'ADMIN' ? '/admin/dashboard' : '/user/dashboard';
  }

  private setUser(user: User): void {
    this.currentUser.set(user);
    try {
      localStorage.setItem(AuthService.STORAGE_KEY, JSON.stringify(user));
    } catch {
      // Mode privat atau storage penuh. Aplikasi tetap jalan, hanya saja
      // guard harus menunggu /auth/me setiap kali halaman dimuat ulang.
    }
  }

  private clearUser(): void {
    this.currentUser.set(null);
    try {
      localStorage.removeItem(AuthService.STORAGE_KEY);
    } catch {
      /* diabaikan, alasannya sama seperti di setUser */
    }
  }
}

function readStoredUser(): User | null {
  try {
    const raw = localStorage.getItem('geosquad.user');
    if (!raw) return null;
    const parsed = JSON.parse(raw) as User;
    // Isi storage bisa saja rusak atau diutak-atik. Yang bentuknya tidak
    // masuk akal dibuang, bukan dipakai apa adanya.
    return parsed?.id && parsed?.role ? parsed : null;
  } catch {
    return null;
  }
}
