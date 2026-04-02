# quickpod-cli

Go CLI for the user-facing QuickPod GPU and CPU platform APIs.

Scope:
- Uses only `quickpod-api` read-only routes and user-safe `quickpod-update-api` routes.
- Excludes internal operational endpoints such as ad hoc pod lifecycle routes and machine reset routes.
- Focuses on discovery, pods, pod clusters, serverless endpoints, templates, machines, storage volumes, account workflows, host stores, and 2FA.

Base API:
- Default base URL: `https://api.quickpod.org`

## Features

- Search rentable or occupied GPU and CPU offers
- Inspect public catalog data like GPU types, pricing, distribution, locations, and host stores
- Log in, sign up, store tokens locally, and inspect the authenticated profile
- List, create, reset, rename, start, stop, restart, destroy, and inspect pods, with `get` and `--wide` views
- List, create, scale, inspect, and operate pod clusters and cluster services
- List, create, inspect, update, delete, and tail logs for serverless endpoints
- List and manage templates, including community toggles
- List machines, inspect contracts, update host listing settings, and use `get` and `--wide` views
- List storage servers, inspect one server, and manage user volumes with `--wide` views
- View account metrics, transactions, affiliations, audit history, and host earnings
- Manage 2FA with email or TOTP flows

## Build

```bash
go mod tidy
go build -o quickpod
```

## Authentication

The CLI stores its auth credential in:

```text
~/.config/quickpod-cli/config.json
```

That credential can be either:
- A bearer token returned by `quickpod auth login`
- A secure API key beginning with `qpk_`

You can also override runtime settings with environment variables:

```bash
export QUICKPOD_BASE_URL=https://api.quickpod.org
export QUICKPOD_TOKEN=...
export QUICKPOD_API_KEY=qpk_...
export QUICKPOD_OUTPUT=json
```

If both `QUICKPOD_TOKEN` and `QUICKPOD_API_KEY` are set, the API key wins.

## Quick Start

Log in interactively:

```bash
./quickpod auth login --email you@example.com
```

If the account has two-factor enabled, the CLI completes the login challenge as well:

```bash
./quickpod auth login --email you@example.com --two-factor-code 123456
```

For email-based 2FA, the first login call triggers the email challenge and then prompts for the code.
For TOTP-based 2FA, the CLI prompts immediately unless `--two-factor-code` is provided.

Google OAuth login:

```bash
./quickpod auth google --print-auth-url --client-id YOUR_GOOGLE_CLIENT_ID --redirect-uri https://your.app/callback
./quickpod auth google --code YOUR_CODE --redirect-uri https://your.app/callback
```

GitHub OAuth login:

```bash
./quickpod auth github --print-auth-url --client-id YOUR_GITHUB_CLIENT_ID --redirect-uri https://your.app/callback
./quickpod auth github --code YOUR_CODE --redirect-uri https://your.app/callback
```

If the OAuth identity does not exist yet, the CLI handles the backend `signup_required` flow and prompts for `user` or `host` when needed.
OAuth login also supports the same two-factor challenge flow as password login.

Store an existing bearer token:

```bash
./quickpod auth set-token --value "$QUICKPOD_TOKEN"
```

Store an existing secure API key:

```bash
./quickpod auth set-api-key --value "$QUICKPOD_API_KEY"
```

You can also pass a credential per invocation:

```bash
./quickpod --token "$QUICKPOD_TOKEN" auth me
./quickpod --api-key "$QUICKPOD_API_KEY" auth me
```

Check your profile:

```bash
./quickpod auth me
```

Use a secure API key against mixed-auth read-only routes:

```bash
./quickpod --api-key "$QUICKPOD_API_KEY" auth me
./quickpod --api-key "$QUICKPOD_API_KEY" pods list --kind gpu
./quickpod --api-key "$QUICKPOD_API_KEY" templates list --scope my --kind gpu
```

Search GPU offers:

```bash
./quickpod search gpu --type A100 --max-hourly 2.5 --verified-only
./quickpod search gpu --sort reliability --desc --limit 10
```

Search CPU offers:

```bash
./quickpod search cpu --max-hourly 0.25 --min-count 8
```

List your pods:

```bash
./quickpod pods list --kind gpu
./quickpod pods list --kind cpu
./quickpod pods get --kind gpu --pod POD_UUID
./quickpod pods list --kind gpu --wide
./quickpod pods logs --kind gpu --pod POD_UUID
```

Create a GPU pod:

```bash
./quickpod pods create \
	--kind gpu \
	--template TEMPLATE_UUID \
	--offer 12345 \
	--disk 50 \
	--name trainer
```

Create a CPU job:

```bash
./quickpod pods create \
	--kind cpu \
	--job \
	--template TEMPLATE_UUID \
	--offer 12345 \
	--disk 20
```

Start or stop a pod:

```bash
./quickpod pods stop --kind gpu --pod POD_UUID
./quickpod pods start --kind gpu --pod POD_UUID
```

List public templates:

```bash
./quickpod templates list --scope public --kind gpu
./quickpod templates list --scope community --kind cpu
```

Save a template from flags:

```bash
./quickpod templates save \
	--kind gpu \
	--name my-template \
	--image-path ghcr.io/acme/image:latest \
	--disk-space 30 \
	--public=false
```

Save a template from JSON:

```bash
./quickpod templates save --file ./template.json
```

List your machines and contracts:

```bash
./quickpod machines list --kind gpu
./quickpod machines get --kind gpu --id 14717
./quickpod machines contracts
```

Work with pod clusters:

```bash
./quickpod clusters list
./quickpod clusters get --id 12
./quickpod clusters create --file ./cluster.json
./quickpod clusters scale --id 12 --replicas 4 --offer-id 101 --offer-id 102
./quickpod clusters services list --cluster-id 12
```

Work with serverless endpoints:

```bash
./quickpod serverless list
./quickpod serverless get --id 9
./quickpod serverless create --file ./endpoint.json
./quickpod serverless logs --id 9 --limit 25
```

Update a GPU machine listing:

```bash
./quickpod machines update-gpu \
	--machine-id 14717 \
	--listed true \
	--min-gpu 1 \
	--max-duration 24 \
	--storage-cost 0.05 \
	--inet-down-cost 0.00 \
	--gpu-price 101=0.79 \
	--gpu-price 102=0.79
```

List storage servers and volumes:

```bash
./quickpod storage servers
./quickpod storage servers get --id 4
./quickpod storage volumes list
./quickpod storage volumes list --wide
./quickpod storage volumes get --id 42
```

Inspect referral activity and account history:

```bash
./quickpod account affiliations
./quickpod account transactions
./quickpod account audit-log
```

Create a storage volume:

```bash
./quickpod storage volumes create \
	--server-id 4 \
	--name datasets \
	--size-gb 250 \
	--allowed-host 10.0.0.10
```

2FA flows:

```bash
./quickpod security 2fa-status
./quickpod security enable-email-2fa
./quickpod security setup-totp
./quickpod security enable-totp --code 123456
./quickpod security disable-2fa
```

## Output Formats

Default output is table-oriented for interactive use.
Most table commands now include operational identifiers like machine IDs, offer IDs, access ranges, service IDs, and qualification state where those fields are available from the APIs.

Switch to JSON per command:

```bash
./quickpod --output json pods list --kind gpu
```

Or globally:

```bash
export QUICKPOD_OUTPUT=json
```

## Command Groups

- `auth`: login, signup, me, logout, token or API-key handling
- `search`: rentable and occupied GPU or CPU offers
- `catalog`: public types, pricing, distribution, locations, host stores
- `pods`: list, history, create, reset, lifecycle, rename
- `templates`: list, save, delete, community flag
- `machines`: list, contracts, listing updates, privileged access
- `storage`: servers and user volumes
- `account`: metrics, history, contact, email check, API key reset, reverify email
- `security`: 2FA status and setup
- `store`: host store listing and upsert

## Notes

- Webhooks, background backend helpers, and other infrastructure-only routes are intentionally excluded.
- Some endpoints return large JSON payloads; use `--output json` when you need the full response.
- When a stored credential starts with `qpk_`, the CLI automatically sends it as `X-API-Key` and `Authorization: ApiKey ...`; all other credentials are sent as bearer tokens.
