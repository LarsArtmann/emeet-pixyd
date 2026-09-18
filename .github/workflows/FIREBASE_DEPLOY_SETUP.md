# Firebase deploy CI — one-time setup checklist (Lars)

Ends the manual-deploy dependency (plan M28 / TODO #133). The deploy job is
already wired in `.github/workflows/website.yml` (job `deploy`) and gated on
the `FIREBASE_SERVICE_ACCOUNT` secret existing — until the secret is added,
the job self-skips and the build-only job runs as today.

## Create the service account (5 minutes)

1. Console: <https://console.firebase.google.com/project/lars-software/settings/serviceaccounts/serviceaccounts>
2. "Generate new private key" on the existing **firebase-adminsdk** account
   (or create a dedicated `github-deploy` service account with role
   **Firebase Hosting Admin** — `roles/firebasehosting.admin` — least
   privilege preferred).
3. Download the JSON key file. Do NOT commit it.

## Add the secret

4. Repo → Settings → Secrets and variables → Actions → **New repository
   secret**
   - Name: `FIREBASE_SERVICE_ACCOUNT` (exact — the workflow reads this name)
   - Value: the FULL contents of the downloaded JSON key file.

## Verify

5. Push any `website/**` change to master (or re-run the last website
   workflow). The `deploy` job must run (not skip), and
   <https://emeet-pixyd.lars.software/> must serve the built content.
6. Rotate: the key file lives only in the secret; delete the local download
   after step 4.

## Rollback / safety

- Deploys are atomic Firebase Hosting releases; rollback is one click in
  Firebase Console → Hosting → Release history.
- The deploy job runs `pnpm run build` fresh and greps dist for the release
  version BEFORE uploading (stale-deploy incident rule), so a wrong-content
  deploy fails the job instead of shipping.
