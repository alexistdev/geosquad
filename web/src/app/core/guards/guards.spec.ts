import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import {
  ActivatedRouteSnapshot,
  GuardResult,
  Router,
  RouterStateSnapshot,
  UrlTree,
} from '@angular/router';
import { provideRouter } from '@angular/router';
import { Observable } from 'rxjs';

import { authGuard } from './auth.guard';
import { roleGuard } from './role.guard';
import { guestGuard } from './guest.guard';
import { AuthService } from '../services/auth.service';
import { User } from '../models/user';

function makeUser(role: 'ADMIN' | 'USER'): User {
  return {
    id: 'u-1',
    fullName: 'Tes',
    email: 'tes@example.com',
    role,
    isSuspended: false,
    createdDate: '2026-09-19T00:00:00Z',
  };
}

/** Menaruh user ke dalam service lewat jalur login yang sebenarnya. */
function login(role: 'ADMIN' | 'USER'): void {
  const auth = TestBed.inject(AuthService);
  const http = TestBed.inject(HttpTestingController);
  auth.login({ email: 'tes@example.com', password: 'x' }).subscribe();
  http.expectOne('/api/v1/auth/login').flush({
    status: true,
    messages: [],
    payload: {
      sessionId: 'abc',
      user: makeUser(role),
      defaultHomeUrl: role === 'ADMIN' ? '/admin/dashboard' : '/user/dashboard',
      expiresIn: 86400,
    },
  });
}

function routeWithRoles(roles: string[]): ActivatedRouteSnapshot {
  return { data: { roles } } as unknown as ActivatedRouteSnapshot;
}

const anyState = { url: '/admin/users' } as RouterStateSnapshot;

describe('guards', () => {
  let http: HttpTestingController;

  beforeEach(() => {
    localStorage.clear();
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    });
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => localStorage.clear());

  describe('authGuard', () => {
    it('meloloskan user yang sudah login tanpa memanggil server', () => {
      login('USER');
      const result = TestBed.runInInjectionContext(() =>
        authGuard(routeWithRoles([]), anyState),
      );

      expect(result).toBeTrue();
      http.verify(); // tidak ada request /auth/me yang menggantung
    });

    it('bertanya ke server saat profil lokal kosong, lalu menolak kalau tidak ada sesi', (done) => {
      // authGuard mengembalikan Observable saat harus bertanya ke server.
      const result = TestBed.runInInjectionContext(() =>
        authGuard(routeWithRoles([]), anyState),
      ) as Observable<GuardResult>;

      result.subscribe((value) => {
        expect(value instanceof UrlTree).toBeTrue();
        expect((value as UrlTree).toString()).toContain('/login');
        done();
      });

      http.expectOne('/api/v1/auth/me').flush(
        { status: false, messages: [] },
        { status: 401, statusText: 'Unauthorized' },
      );
    });
  });

  describe('roleGuard', () => {
    it('meloloskan peran yang cocok', () => {
      login('ADMIN');
      const result = TestBed.runInInjectionContext(() =>
        roleGuard(routeWithRoles(['ADMIN']), anyState),
      );
      expect(result).toBeTrue();
    });

    // Peran yang salah dilempar ke dashboard-nya sendiri, bukan ke login:
    // user ini sah, cuma salah alamat.
    it('melempar peran yang tidak cocok ke dashboard miliknya', () => {
      login('USER');
      const result = TestBed.runInInjectionContext(() =>
        roleGuard(routeWithRoles(['ADMIN']), anyState),
      ) as UrlTree;

      expect(result instanceof UrlTree).toBeTrue();
      expect(result.toString()).toBe('/user/dashboard');
    });

    it('melempar ke login kalau tidak ada user sama sekali', () => {
      const result = TestBed.runInInjectionContext(() =>
        roleGuard(routeWithRoles(['ADMIN']), anyState),
      ) as UrlTree;

      expect(result.toString()).toBe('/login');
    });
  });

  describe('guestGuard', () => {
    it('meloloskan tamu', () => {
      const result = TestBed.runInInjectionContext(() => guestGuard(routeWithRoles([]), anyState));
      expect(result).toBeTrue();
    });

    it('melempar user yang sudah login ke dashboard-nya', () => {
      login('ADMIN');
      const result = TestBed.runInInjectionContext(() =>
        guestGuard(routeWithRoles([]), anyState),
      ) as UrlTree;

      expect(result.toString()).toBe('/admin/dashboard');
    });
  });
});

describe('Router tersedia untuk guard', () => {
  it('menginjeksi Router tanpa error', () => {
    TestBed.configureTestingModule({ providers: [provideRouter([])] });
    expect(TestBed.inject(Router)).toBeTruthy();
  });
});
