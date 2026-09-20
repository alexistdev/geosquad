import { Injectable, signal } from '@angular/core';

type Mode = 'light' | 'dark';
type SidebarSize = 'lg' | 'sm';

/**
 * Mengatur atribut data-* di elemen <html> yang dipakai Velzon untuk
 * menentukan tema dan ukuran sidebar.
 *
 * Velzon membawa app.js yang melakukan hal serupa dengan memanipulasi DOM
 * secara langsung dan memasang listener sendiri. Itu tidak dipakai: Angular
 * yang memiliki DOM di sini, dan dua pihak yang sama-sama mengubahnya akan
 * saling menimpa. Yang diambil hanya kontraknya -- nama atribut dan nilainya.
 */
@Injectable({ providedIn: 'root' })
export class ThemeService {
  private static readonly MODE_KEY = 'geosquad.theme';
  private static readonly SIDEBAR_KEY = 'geosquad.sidebar';

  private readonly root = document.documentElement;

  readonly mode = signal<Mode>(read<Mode>(ThemeService.MODE_KEY, preferredMode()));
  readonly sidebarSize = signal<SidebarSize>(read<SidebarSize>(ThemeService.SIDEBAR_KEY, 'lg'));

  constructor() {
    this.apply();
  }

  toggleMode(): void {
    this.mode.set(this.mode() === 'light' ? 'dark' : 'light');
    store(ThemeService.MODE_KEY, this.mode());
    this.apply();
  }

  /** Menciutkan sidebar jadi ikon saja, atau mengembalikannya. */
  toggleSidebar(): void {
    this.sidebarSize.set(this.sidebarSize() === 'lg' ? 'sm' : 'lg');
    store(ThemeService.SIDEBAR_KEY, this.sidebarSize());
    this.apply();
  }

  /**
   * Di layar sempit sidebar tampil sebagai panel melayang, bukan menciut.
   * Velzon menandainya dengan kelas pada <body>.
   */
  toggleMobileSidebar(): void {
    document.body.classList.toggle('vertical-sidebar-enable');
  }

  closeMobileSidebar(): void {
    document.body.classList.remove('vertical-sidebar-enable');
  }

  private apply(): void {
    const mode = this.mode();
    this.root.setAttribute('data-bs-theme', mode);
    // Sidebar sengaja selalu gelap, di mode terang maupun gelap. Kontras
    // dengan area isi membuat batas navigasi terbaca tanpa perlu garis.
    this.root.setAttribute('data-sidebar', 'dark');
    // Topbar dibiarkan "light" di kedua mode. Nilai "dark" di Velzon berarti
    // topbar berwarna primary, bukan gelap -- sebuah pita biru yang menabrak
    // sisa halaman.
    this.root.setAttribute('data-topbar', 'light');
    this.root.setAttribute('data-sidebar-size', this.sidebarSize());
  }
}

function preferredMode(): Mode {
  try {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  } catch {
    return 'light';
  }
}

function read<T extends string>(key: string, fallback: T): T {
  try {
    return (localStorage.getItem(key) as T) || fallback;
  } catch {
    // Mode privat atau storage diblokir. Preferensi tidak diingat, tapi
    // aplikasi tetap tampil benar.
    return fallback;
  }
}

function store(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
  } catch {
    /* alasannya sama seperti read() */
  }
}
