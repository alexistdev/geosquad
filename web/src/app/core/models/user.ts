export type Role = 'ADMIN' | 'USER';

export interface User {
  id: string;
  fullName: string;
  email: string;
  role: Role;
  isSuspended: boolean;
  createdDate: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  sessionId: string;
  user: User;
  defaultHomeUrl: string;
  expiresIn: number;
}

export interface RegisterRequest {
  fullName: string;
  email: string;
  password: string;
}

export interface CreateUserRequest {
  fullName: string;
  email: string;
  password: string;
  role: Role;
}

export interface UpdateUserRequest {
  fullName?: string;
  role?: Role;
  isSuspended?: boolean;
}

export interface ChangePasswordRequest {
  currentPassword: string;
  newPassword: string;
}
