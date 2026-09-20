import { Component, inject, signal } from '@angular/core';
import { NavigationEnd, Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { filter } from 'rxjs';

import { AuthService } from '../../core/services/auth.service';
import { ThemeService } from '../../core/services/theme.service';

/** Kerangka halaman setelah login: sidebar Velzon, topbar, lalu isi rutenya. */
@Component({
  selector: 'app-shell',
  imports: [RouterOutlet, RouterLink, RouterLinkActive],
  templateUrl: './shell.html',
})
export class Shell {
  private readonly router = inject(Router);
  protected readonly auth = inject(AuthService);
  protected readonly theme = inject(ThemeService);

  /**
   * Dropdown user di topbar, dikelola Angular sendiri.
   *
   * Velzon mengandalkan Bootstrap JS untuk ini. Bootstrap JS tidak dimuat:
   * satu dropdown tidak sebanding dengan menambah library yang memasang
   * listener di elemen yang dikendalikan Angular.
   */
  protected readonly menuOpen = signal(false);

  constructor() {
    // Di layar sempit sidebar menutupi halaman. Tanpa ini, user mendarat di
    // halaman baru yang masih tertutup menu.
    this.router.events
      .pipe(filter((event) => event instanceof NavigationEnd))
      .subscribe(() => {
        this.theme.closeMobileSidebar();
        this.menuOpen.set(false);
      });
  }

  protected toggleSidebar(): void {
    // Velzon memakai dua mekanisme berbeda: di layar lebar sidebar menciut
    // jadi ikon, di layar sempit ia muncul sebagai panel melayang.
    if (window.innerWidth < 768) {
      this.theme.toggleMobileSidebar();
    } else {
      this.theme.toggleSidebar();
    }
  }

  protected logout(): void {
    this.menuOpen.set(false);
    this.auth.logout().subscribe(() => this.router.navigate(['/login']));
  }

  /** Inisial untuk avatar, karena tidak ada foto profil. */
  protected initials(): string {
    const name = this.auth.user()?.fullName ?? '';
    return name
      .split(/\s+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((part) => part[0]!.toUpperCase())
      .join('');
  }
}
