# Dad Pack License Integration

This document describes the Dad Pack license integration with LemonSqueezy.

## Overview

MC Dad Server integrates with [LemonSqueezy](https://lemonsqueezy.com) for one-time license validation to unlock Dad Pack premium features.

## Dad Pack Features

When you have a valid Dad Pack license, you get access to:

- **GriefPrevention** — Auto-configured so kids' builds are protected
- **Dynmap** — Web-based live map (show kids their world on a tablet)
- **Web Dashboard** — Simple status page you can bookmark
- **Dad's Guide PDF** — Non-technical guide to being a Minecraft server admin

> **Note**: Dad Pack features are currently in development and will be available in a future update.

## License Commands

### Activate a License

Activate your Dad Pack license for this server:

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

## Using License During Install

You can provide a license key during installation:

```bash
mc-dad-server install --license YOUR-LICENSE-KEY
```

The installer will:
1. Validate the license with LemonSqueezy
2. If valid, enable Dad Pack features
3. Store the license for future use

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
```

### Testing

Run license tests:

```bash
go test ./internal/license/...
```

### Adding New Features

To add a new Dad Pack feature:

1. Update `internal/dadpack/dadpack.go`:
   - Add feature to `Features` struct
   - Implement installation in `InstallFeatures`
   - Update `GetAvailableFeatures`

2. Update the UI summary in `internal/ui/summary.go`

3. Test the feature with a valid license
