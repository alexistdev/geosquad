import { Component, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { DatePipe } from '@angular/common';

import { UserService } from '../../core/services/user.service';
import { AuthService } from '../../core/services/auth.service';
import { CreateUserRequest, Role, User } from '../../core/models/user';

@Component({
  selector: 'app-admin-users',
  imports: [FormsModule, DatePipe],
  templateUrl: './admin-users.html',
})
export class AdminUsers implements OnInit {
  private readonly users = inject(UserService);
  protected readonly auth = inject(AuthService);

  protected readonly items = signal<User[]>([]);
  protected readonly total = signal(0);
  protected readonly loading = signal(true);
  protected readonly error = signal('');
  protected readonly notice = signal('');

  protected search = '';
  protected roleFilter: Role | '' = '';

  protected readonly showForm = signal(false);
  protected readonly saving = signal(false);
  protected draft: CreateUserRequest = emptyDraft();

  ngOnInit(): void {
    this.load();
  }

  protected load(): void {
    this.loading.set(true);
    this.users.list({ perPage: 50, search: this.search.trim(), role: this.roleFilter }).subscribe({
      next: (page) => {
        this.items.set(page.items);
        this.total.set(page.meta.total);
        this.loading.set(false);
      },
      error: (err: Error) => {
        this.loading.set(false);
        this.error.set(err.message);
      },
    });
  }

  protected openForm(): void {
    this.draft = emptyDraft();
    this.error.set('');
    this.showForm.set(true);
  }

  protected create(): void {
    this.error.set('');
    this.saving.set(true);

    this.users.create({ ...this.draft, email: this.draft.email.trim() }).subscribe({
      next: (user) => {
        this.saving.set(false);
        this.showForm.set(false);
        this.notice.set(`User ${user.email} berhasil dibuat.`);
        this.load();
      },
      error: (err: Error) => {
        this.saving.set(false);
        this.error.set(err.message);
      },
    });
  }

  protected changeRole(user: User, role: Role): void {
    if (role === user.role) return;
    this.apply(user, { role }, `Peran ${user.email} diubah jadi ${role}.`);
  }

  protected toggleSuspend(user: User): void {
    const next = !user.isSuspended;
    const verb = next ? 'dinonaktifkan' : 'diaktifkan';
    this.apply(user, { isSuspended: next }, `${user.email} ${verb}.`);
  }

  protected remove(user: User): void {
    if (!confirm(`Hapus ${user.email}? Semua sesinya akan dicabut.`)) return;

    this.error.set('');
    this.users.delete(user.id).subscribe({
      next: () => {
        this.notice.set(`${user.email} dihapus.`);
        this.load();
      },
      error: (err: Error) => this.error.set(err.message),
    });
  }

  /**
   * Server memegang aturan yang tidak boleh dilanggar -- admin aktif terakhir
   * tidak bisa diturunkan, akun sendiri tidak bisa dinonaktifkan. Frontend
   * tidak menduplikasi aturan itu; ia cukup menampilkan penolakannya. Menyalin
   * aturan ke sini hanya menciptakan dua tempat yang bisa berbeda pendapat.
   */
  private apply(user: User, patch: Parameters<UserService['update']>[1], success: string): void {
    this.error.set('');
    this.notice.set('');

    this.users.update(user.id, patch).subscribe({
      next: () => {
        this.notice.set(success);
        this.load();
      },
      error: (err: Error) => {
        this.error.set(err.message);
        // Muat ulang supaya kontrol kembali ke keadaan sebenarnya di server,
        // bukan tertinggal pada pilihan yang barusan ditolak.
        this.load();
      },
    });
  }

  protected isSelf(user: User): boolean {
    return this.auth.user()?.id === user.id;
  }
}

function emptyDraft(): CreateUserRequest {
  return { fullName: '', email: '', password: '', role: 'USER' };
}
