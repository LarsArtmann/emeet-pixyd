# Firebase Hosting deploy setup

The website (`emeet-pixyd.lars.software`) deploys automatically from
`.github/workflows/website.yml` on every push to `master` that touches
`website/**`. This document records how the CI credentials were created and
how to rotate or replace them.

**Status: done (2026-09-19).** The `FIREBASE_SERVICE_ACCOUNT` repository secret
is set, and the `deploy` job no longer self-skips. The workflow also has a
manual `workflow_dispatch` trigger for re-deploys and verification.

## What CI uses

| Item        | Value                                                                 |
| ----------- | --------------------------------------------------------------------- |
| GCP project | `lars-software`                                                       |
| Service account | `github-website-deploy@lars-software.iam.gserviceaccount.com`      |
| Role        | `roles/firebasehosting.admin` (already granted)                       |
| GitHub secret | `FIREBASE_SERVICE_ACCOUNT` (the service account's JSON key)          |
| Hosting target | `emeet-pixyd` (Firebase project `lars-software`)                    |

The deploy job writes the secret to a temporary file, exports
`GOOGLE_APPLICATION_CREDENTIALS`, and runs
`npx --yes firebase-tools@latest deploy --only hosting:emeet-pixyd --project lars-software`.
Before uploading it rebuilds the site and greps the freshly built changelog for
the newest documented section (the stale-deploy guard).

## Rotation (run as an owner of `lars-software`, e.g. `lartyhd@gmail.com`)

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
firebase deploy --only hosting:emeet-pixyd --project lars-software
```
