import { socket } from "../ws";

export function todayKey() {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`;
}

export function isAdminRoute() {
  return window.location.hash === "#admin" || window.location.pathname.replace(/\/$/, "").endsWith("/admin");
}

export function ask<T>(event: string, payload: unknown): Promise<T> {
  return new Promise((resolve, reject) => {
    socket.emit(event, payload, (response: T & { error?: string }) => {
      if (response?.error) reject(new Error(response.error));
      else resolve(response);
    });
  });
}
