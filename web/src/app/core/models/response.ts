/*
 * Bentuk response yang dipakai seluruh API, sama seperti geobill dan
 * geolicense: { status, messages, payload }.
 */

export interface BaseResponse<T> {
  status: boolean;
  messages: string[];
  payload?: T;
}

export interface PageMeta {
  page: number;
  perPage: number;
  total: number;
  totalPages: number;
}

export interface Paged<T> {
  items: T[];
  meta: PageMeta;
}
