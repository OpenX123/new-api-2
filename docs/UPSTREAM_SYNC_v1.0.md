# Upstream Sync — v1.0 (97 commits)

**Merged**: 2026-05-13
**Base**: `af27b875` (last sync)
**Head**: `18282e61` (upstream/main) → fork `a8903283` (merge commit)
**Range**: `af27b875..18282e61`
**Counts**: 30 feat · 34 fix · 2 refactor · 4 chore · 1 docs · 12 other (including the v1.0 launch commit `a42b3976`)

---

## TL;DR — Highlights

- 🚀 **v1.0 next-generation frontend** (`a42b3976`) — brand-new `web/default/` built on React 19 + TanStack (Router/Query/Table/Virtual) + Base UI + Tailwind + Bun + Rsbuild; the old frontend is preserved verbatim at `web/classic/`. A user-facing toggle (`feat(ui): add classic frontend switch`, `dac55f0f`) lets each user pick which UI to use.
- 💳 **Payment hardening** — `PaymentProvider` field added to TopUp to prevent cross-gateway callback attacks (`a7c38ec8`); `payment_compliance.go` + `feat: require compliance confirmation for paid features` (`0526a226`); top_up_link moved from `/api/status` to topup info API (`5c793d79`); wallet gateway flags now from `topupInfo` (`3057f04a`).
- 📊 **Model performance metrics** — collection (`9acf5fec`) + dashboard badges (`e8cfb546`) + health panel (`03d53732`); upstream Request-ID is tracked and response-header override is blocked (`aa56667b`).
- 🔐 **OIDC + multi-provider OAuth** — Discord / GitHub / LinuxDo controllers added (`controller/discord.go`, `github.go`, `linuxdo.go`, `oidc.go`); `user.created_at` and `last_login_at` added (`02aacb38`).
- ⚖️ **Licensing** — `LICENSE` / `NOTICE` / `THIRD-PARTY-LICENSES.md` shipped inside the Docker image (`543cc64e`); `.dockerignore` updated so `THIRD-PARTY-LICENSES.md` is included (`5fa103fa`).
- 🧪 **Tests** — `common/json_test.go`, `controller/channel_upstream_update_test.go`, `controller/model_list_test.go` added.
- 🎨 **Affinity error code** — switched to `403 + MsgDistributorAffinityChannelDisabled` (`35530722`) — *conflicted with our 503; we adopted upstream.*
- 🧹 **CI** — `docker-build.yml` reworked; ARM64-only workflow removed; release workflow updated.

---

## Conflicts resolved during merge

| File | Resolution | Why |
| --- | --- | --- |
| `middleware/distributor.go` | adopt upstream (`403 + MsgDistributorAffinityChannelDisabled`) | More precise i18n key; matches upstream tests and docs. |
| `model/topup.go` | **merge both** — keep our `Username` field + `fillTopUpUsernames()` helper, add upstream's `PaymentProvider` field | Username comes from our admin-side topup listing patch; PaymentProvider is the security fix from upstream. |
| `web/classic/src/pages/Home/index.jsx` | adopt upstream | `web/classic/` is the *new* location of the legacy UI; we follow upstream there. |
| `web/classic/src/components/table/users/modals/BatchDeleteUserModal.jsx` | accept relocation from `web/src/` → `web/classic/` | Mechanical move; content unchanged. |
| `.github/workflows/docker-build.yml` | adopt upstream | Upstream rewrote the workflow; our customizations didn't apply. |
| `electron/package.json` + `package-lock.json` | adopt upstream | Lockfile drift only. |

---

## What changed where

### Backend (Go)

- `controller/discord.go`, `github.go`, `linuxdo.go`, `oidc.go` — new OAuth providers.
- `controller/payment_compliance.go` — compliance confirmation gate for paid features.
- `controller/channel_upstream_update.go` + `_test.go` — upstream-driven channel update logic + tests.
- `controller/model_list_test.go` — new model-list endpoint tests.
- `common/json.go` + `common/json_test.go` — JSON helper hardening.
- `model/topup.go` — `PaymentProvider` field + cross-gateway callback guards.
- `service/` — performance metrics collection, Request-ID tracking.

### Frontend

- **New**: `web/default/` — React 19 + Base UI + TanStack + Rsbuild + Bun. See `web/default/AGENTS.md` for the in-tree spec.
- **Renamed**: `web/src/` → `web/classic/` (old Semi-Design UI, kept for backwards compat).
- `main.go` now embeds both `web/default/dist` and `web/classic/dist`; a runtime switch picks which one to serve.

### Infra / Docs

- `Dockerfile.dev` added for local dev.
- `NOTICE`, `THIRD-PARTY-LICENSES.md`, `.github/FUNDING.yml`, `.github/SECURITY.md` updated.
- `README.en.md` added; other README locales refreshed.

---

## Local preservation (our 47 ahead commits)

All preserved on top of the merge:

- `model/topup.go` — `Username` field + `fillTopUpUsernames()` (admin-side topup listing).
- `.agents/skills/` — `classic-to-default-sync`, `i18n-translate`, `shadcn-ui`, `vercel-react-best-practices` skill packs.
- Any other yiranxiaohui-only patches before `af27b875`.

---

## Verification

- `go build ./...` — ✅ clean (Go 1.25.1).
- Frontend build — see `docs/UPSTREAM_SYNC_v1.0.md` follow-up (verified separately after merge).
- Conflicts — 0 remaining post-merge.
- Upstream parity — `git rev-list --left-right --count upstream/main...main` = `0 49` (we are ahead by 49 only with our originals).

---

## Follow-ups

- [ ] Add unit test around `fillTopUpUsernames` to lock in our Username-merge contract against future upstream churn on `TopUp`.
- [ ] When deploying, run `web/default` build first (Rsbuild) before `docker build`.
- [ ] Consider squashing some of our `.agents/skills/*` history into a single skills commit before next sync.
