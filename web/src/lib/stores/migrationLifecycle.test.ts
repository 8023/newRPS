import { readFileSync } from "node:fs";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../rpc", () => ({
  ask: vi.fn(),
  isAdminRoute: () => false
}));

vi.mock("./uiStore.svelte", () => ({
  uiStore: { notify: vi.fn() }
}));

import { adminStore } from "./adminStore.svelte";
import { routerStore } from "./routerStore.svelte";
import { ask } from "../rpc";

describe("Svelte migration lifecycle wiring", () => {
  beforeEach(() => {
    let hash = "";
    const location = {} as Location;
    Object.defineProperty(location, "hash", {
      get: () => hash,
      set: (next: string) => {
        hash = next && !next.startsWith("#") ? `#${next}` : next;
      }
    });
    vi.stubGlobal("window", {
      location,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn()
    });
    adminStore.resetSession();
    vi.mocked(ask).mockReset();
    routerStore.keepContribute = false;
    routerStore.goto("lobby");
  });

  it("returns from admin to the source view and clears the admin hash", () => {
    routerStore.goto("room");
    routerStore.openAdmin();

    expect(routerStore.view).toBe("admin");
    expect(window.location.hash).toBe("#admin");

    routerStore.leaveAdmin(true, true);

    expect(routerStore.view).toBe("room");
    expect(window.location.hash).toBe("");
  });

  it("clears credentials, dirty drafts and local admin UI state on unmount", () => {
    adminStore.password = "secret";
    adminStore.logged = true;
    adminStore.draft = {} as never;
    adminStore.dirty = true;
    adminStore.serverConfigChanged = true;
    adminStore.activeSection = "rooms";
    adminStore.factionSearch = "keyword";
    adminStore.announcementMessage = "message";
    adminStore.adminPlayersTotal = 12;
    adminStore.contributionCounts = { pending: 3 };

    adminStore.resetSession();

    expect(adminStore.password).toBe("");
    expect(adminStore.logged).toBe(false);
    expect(adminStore.draft).toBeNull();
    expect(adminStore.dirty).toBe(false);
    expect(adminStore.serverConfigChanged).toBe(false);
    expect(adminStore.activeSection).toBe("site");
    expect(adminStore.factionSearch).toBe("");
    expect(adminStore.announcementMessage).toBe("");
    expect(adminStore.adminPlayersTotal).toBe(0);
    expect(adminStore.contributionCounts).toEqual({ pending: 0 });
  });

  it("does not let an in-flight login restore a closed admin session", async () => {
    let resolveLogin!: (value: unknown) => void;
    vi.mocked(ask).mockReturnValueOnce(new Promise((resolve) => {
      resolveLogin = resolve;
    }));
    adminStore.password = "secret";

    const login = adminStore.login();
    adminStore.resetSession();
    resolveLogin({});
    await login;

    expect(adminStore.logged).toBe(false);
    expect(adminStore.password).toBe("");
  });

  it("keeps pageview effects and the admin lazy boundary in App", () => {
    const source = readFileSync(new URL("../../App.svelte", import.meta.url), "utf8");

    expect(source).toContain('trackPageview(routerStore.view)');
    for (const view of ["profile", "leaderboard", "about", "help"]) {
      expect(source).toContain(`trackPageview("${view}")`);
    }
    expect(source).toContain('import("./ui/admin/AdminPanel.svelte")');
    expect(source).not.toMatch(/^\s*import AdminPanel from/m);
  });

  it("keeps admin exit/reset wiring and stores chat separators as source text", () => {
    const panelSource = readFileSync(new URL("../../ui/admin/AdminPanel.svelte", import.meta.url), "utf8");
    const helperSource = readFileSync(new URL("../adminHelpers.ts", import.meta.url));

    expect(panelSource).toContain("routerStore.leaveAdmin(Boolean(sessionStore.me), Boolean(sessionStore.room))");
    expect(panelSource).toContain("onDestroy(() => adminStore.resetSession())");
    expect(helperSource.includes(0)).toBe(false);
    expect(helperSource.toString("utf8")).toContain("\\u0000");
  });
});
