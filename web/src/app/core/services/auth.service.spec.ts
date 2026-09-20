import { TestBed } from '@angular/core/testing';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideRouter } from '@angular/router';

import { AuthService } from './auth.service';
import { User } from '../models/user';

const ADMIN: User = {
  id: 'u-1',
  fullName: 'Administrator',
  email: 'admin@geosquad.local',
  role: 'ADMIN',
  isSuspended: false,
  createdDate: '2026-09-19T00:00:00Z',
};

describe('AuthService', () => {
  let service: AuthService;
  let http: HttpTestingController;

  beforeEach(() => {
    localStorage.clear();
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    });
    service = TestBed.inject(AuthService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    http.verify();
    localStorage.clear();
  });

  it('mulai dalam keadaan belum login', () => {
    expect(service.isLoggedIn()).toBeFalse();
    expect(service.user()).toBeNull();
  });

  it('menyimpan profil setelah login berhasil', () => {
    service.login({ email: ADMIN.email, password: 'password' }).subscribe();

    http.expectOne('/api/v1/auth/login').flush({
      status: true,
      messages: ['Login berhasil.'],
      payload: { sessionId: 'abc', user: ADMIN, defaultHomeUrl: '/admin/dashboard', expiresIn: 86400 },
    });

    expect(service.isLoggedIn()).toBeTrue();
    expect(service.isAdmin()).toBeTrue();
    expect(service.user()?.email).toBe(ADMIN.email);
  });

  // Sesi dipegang cookie HttpOnly, jadi tidak ada token yang boleh menyentuh
  // localStorage. Kalau suatu saat ada yang menaruhnya di sana, test ini gagal.
  it('tidak menyimpan sessionId di localStorage', () => {
    service.login({ email: ADMIN.email, password: 'password' }).subscribe();
    http.expectOne('/api/v1/auth/login').flush({
      status: true,
      messages: [],
      payload: { sessionId: 'rahasia-sekali', user: ADMIN, defaultHomeUrl: '/admin/dashboard', expiresIn: 86400 },
    });

    expect(JSON.stringify(localStorage)).not.toContain('rahasia-sekali');
  });

  it('membersihkan profil saat logout', () => {
    service.login({ email: ADMIN.email, password: 'password' }).subscribe();
    http.expectOne('/api/v1/auth/login').flush({
      status: true,
      messages: [],
      payload: { sessionId: 'abc', user: ADMIN, defaultHomeUrl: '/admin/dashboard', expiresIn: 86400 },
    });

    service.logout().subscribe();
    http.expectOne('/api/v1/auth/logout').flush({ status: true, messages: [] });

    expect(service.isLoggedIn()).toBeFalse();
    expect(localStorage.getItem('geosquad.user')).toBeNull();
  });

  // Ganti password mencabut semua sesi di server, termasuk sesi ini. Status
  // lokal harus ikut bersih supaya UI tidak mengira dirinya masih login.
  it('membersihkan profil setelah ganti password', () => {
    service.login({ email: ADMIN.email, password: 'password' }).subscribe();
    http.expectOne('/api/v1/auth/login').flush({
      status: true,
      messages: [],
      payload: { sessionId: 'abc', user: ADMIN, defaultHomeUrl: '/admin/dashboard', expiresIn: 86400 },
    });

    service.changePassword({ currentPassword: 'password', newPassword: 'password-baru' }).subscribe();
    http.expectOne('/api/v1/auth/change-password').flush({ status: true, messages: [] });

    expect(service.isLoggedIn()).toBeFalse();
  });

  it('refresh yang gagal membuat status jadi belum login', () => {
    service.refresh().subscribe();
    http.expectOne('/api/v1/auth/me').flush(
      { status: false, messages: ['Sesi tidak ditemukan.'] },
      { status: 401, statusText: 'Unauthorized' },
    );

    expect(service.isLoggedIn()).toBeFalse();
  });

  it('mengarahkan tiap peran ke dashboard-nya', () => {
    expect(service.homeUrlFor('ADMIN')).toBe('/admin/dashboard');
    expect(service.homeUrlFor('USER')).toBe('/user/dashboard');
    expect(service.homeUrlFor(undefined)).toBe('/user/dashboard');
  });

  it('mengabaikan profil tersimpan yang bentuknya rusak', () => {
    localStorage.setItem('geosquad.user', '{"bukan":"user"}');
    TestBed.resetTestingModule();
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    });
    const fresh = TestBed.inject(AuthService);

    expect(fresh.isLoggedIn()).toBeFalse();
  });
});
