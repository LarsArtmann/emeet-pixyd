# Firebase Hosting deploy setup

The website (`emeet-pixyd.lars.software`) deploys automatically from
`.github/workflows/website.yml` on every push to `master` that touches
`website/**`. This document records how the CI credentials were created and
how to rotate or replace them.

**Status: done (2026-09-19).** The `FIREBASE_SERVICE_ACCOUNT` repository secret
is set, and the `deploy` job no longer self-skips. The workflow also has a
manual `workflow_dispatch` trigger for re-deploys and verification.

## What CI uses

| Item            | Value                                                         |
| --------------- | ------------------------------------------------------------- |
| GCP project     | `lars-software`                                               |
| Service account | `github-website-deploy@lars-software.iam.gserviceaccount.com` |
| Role            | `roles/firebasehosting.admin` (already granted)               |
| GitHub secret   | `FIREBASE_SERVICE_ACCOUNT` (the service account's JSON key)   |
| Hosting target  | `emeet-pixyd` (Firebase project `lars-software`)              |
| Hosting site id | `emeet-pixyd` (same as the target; needed for rollback)        |
| firebase-tools  | `15.35.0` (pinned in `website.yml`; bump deliberately)         |

The deploy job writes the secret to a temporary file, exports
`GOOGLE_APPLICATION_CREDENTIALS`, and runs
`npx --yes firebase-tools@15.35.0 deploy --only hosting:emeet-pixyd --project lars-software`.
Before uploading it rebuilds the site and greps the freshly built changelog for
the newest documented section (the stale-deploy guard). After uploading it
smoke-tests the live URL and writes the release URL to the job summary.

When bumping the `firebase-tools` pin, update ALL THREE places together: the
`Deploy to Firebase Hosting` step, the `Cache firebase-tools` cache key
(`npx-firebase-tools-<version>`), and the reference above.

## Rotation (run as an owner of `lars-software`, e.g. `lartyhd@gmail.com`)

**Key in use (2026-11-05):** `32c6a1484d9f2521f9b33dbf05b76f9ca343d2d9`
(created 2026-09-19 — the setup date above; verify with `gh secret list` and
the key list below before deleting anything).

**Cadence:** rotate at least every 12 months (next due **2027-09-19**) and
immediately on any suspected leak (secret accidentally committed, a
maintainer leaving, an unexpected deploy in the Hosting release history).
Set a calendar reminder — nothing else tracks this. The service account is
shared across the `lars.software` sites; each site's key carries its own
cadence, so only rotate THIS repository's key here. (Fleet note: one other
site's key `de6b2438…` is the only one with a hard expiry, 2028-10-23 — not
ours to manage.)

```bash
KEYFILE=$(mktemp /tmp/emeet-pixyd-sa-XXXXXX.json) && chmod 600 "$KEYFILE"

gcloud iam service-accounts keys create "$KEYFILE" \
  --iam-account=github-website-deploy@lars-software.iam.gserviceaccount.com \
  --project=lars-software \
  --account=lartyhd@gmail.com

gh secret set FIREBASE_SERVICE_ACCOUNT \
  --repo LarsArtmann/emeet-pixyd < "$KEYFILE"

shred -u "$KEYFILE"
```

Then list and retire the old key:

```bash
gcloud iam service-accounts keys list \
  --iam-account=github-website-deploy@lars-software.iam.gserviceaccount.com \
  --project=lars-software --account=lartyhd@gmail.com

gcloud iam service-accounts keys delete <OLD_KEY_ID> \
  --iam-account=github-website-deploy@lars-software.iam.gserviceaccount.com \
  --project=lars-software --account=lartyhd@gmail.com
```

The service account is shared with the other `lars.software` websites. Do not
delete it; only rotate this repository's key material.

## Verify

```bash
gh workflow run website.yml --repo LarsArtmann/emeet-pixyd
gh run watch --repo LarsArtmann/emeet-pixyd
```

The `build` job must be green and the `deploy` job must run (not skip). A
successful deploy prints the hosting release URL for
`https://emeet-pixyd.lars.software`.

## Manual fallback (no CI)

```bash
cd website
nix shell nixpkgs#nodejs -c pnpm run build
npx --yes firebase-tools@15.35.0 deploy --only hosting:emeet-pixyd --project lars-software
```

## Rollback (bad release live)

Firebase Hosting keeps release history; rollback re-activates the previous
release atomically (no rebuild needed):

```bash
# List recent releases with their timestamps/versions
npx --yes firebase-tools@15.35.0 hosting:releases:list \
  --site emeet-pixyd --project lars-software

# Re-activate the previous release
npx --yes firebase-tools@15.35.0 hosting:rollback \
  --site emeet-pixyd --project lars-software
```

Verify with the smoke check the CI deploy job uses:

```bash
curl -fsS https://emeet-pixyd.lars.software/ | grep -i "emeet pixy"
```

If a bad CODE change (not just a bad deploy) is live, fix forward: revert the
commit, let CI redeploy — rollback only masks the content, the next push
redeploys the same code.
