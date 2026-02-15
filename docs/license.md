# License Management

This document describes the MC Dad Server license integration with LemonSqueezy.

## Overview

MC Dad Server is **free and fully functional** — all features work without a license.

A license is an optional way to support the project. Licensed users get:

- **Clean output** — no nag messages after the 7-day grace period
- **"Licensed to [name]" badge** — shown in status and install summary

The license system integrates with [LemonSqueezy](https://lemonsqueezy.com) for one-time license validation.

## How It Works

1. **First 7 days** — no nag messages (grace period)
2. **After 7 days** — a short nag appears at the end of `install`, `start`, `stop`, and `status` output
3. **With a license** — nag disappears, "Licensed to [name]" badge appears

Everything works the same in all three states. No features are locked.

## License Commands

### Activate a License

Activate your license for this server:

```bash
mc-dad-server activate-license --key YOUR-LICENSE-KEY
```

This will:
- Validate the license with LemonSqueezy
- Activate it for this server instance
- Save it locally for future use
- Consume one activation from your license

You can optionally specify a custom instance name:

```bash
mc-dad-server activate-license --key YOUR-LICENSE-KEY --name "my-server"
```

### Validate a License

Check if your license is valid:

```bash
mc-dad-server validate-license
```

This will check the currently stored license. To validate a different license:

```bash
mc-dad-server validate-license --key YOUR-LICENSE-KEY
```

### Deactivate a License

Free up an activation so you can use it on another server:

```bash
mc-dad-server deactivate-license
```

This will:
- Deactivate the license for this server
- Remove the stored license file
- Free up one activation slot

## License Storage

Licenses are stored locally in your server directory:

```
~/minecraft-server/.license
```

This file contains:
- The license key
- Instance ID and name
- Last validation timestamp
- Cached validation response (for offline use)

**Security**: The license file is created with `0600` permissions (owner read/write only).

The install timestamp is tracked separately in `.mc-dad-installed` for grace period calculation.

## Offline Validation

The license manager uses a 24-hour cache for validation responses. This means:

- If you've validated in the last 24 hours, the cached response is used
- If offline, the cached response is used (if available)
- This prevents issues if LemonSqueezy is temporarily unreachable

## API Integration

The license system uses the LemonSqueezy License API:

- **Validate**: `POST https://api.lemonsqueezy.com/v1/licenses/validate`
- **Activate**: `POST https://api.lemonsqueezy.com/v1/licenses/activate`
- **Deactivate**: `POST https://api.lemonsqueezy.com/v1/licenses/deactivate`

No API key is required - license validation is public.

## Troubleshooting

### License validation fails

1. Check your internet connection
2. Verify the license key is correct
3. Check if the license is expired or disabled
4. Ensure you haven't exceeded the activation limit

### Reset a license

If you need to start fresh:

```bash
# Deactivate the current license
mc-dad-server deactivate-license

# Remove the license file (if deactivation failed)
rm ~/minecraft-server/.license

# Activate with a new key
mc-dad-server activate-license --key YOUR-NEW-LICENSE-KEY
```

## Development

### Package Structure

```
internal/license/
├── types.go       # License data structures
├── client.go      # LemonSqueezy API client
├── manager.go     # License validation and storage
└── types_test.go  # Unit tests

internal/nag/
└── nag.go         # Grace period, status resolution, nag display
```

### Testing

Run license tests:

```bash
go test ./internal/license/...
```
