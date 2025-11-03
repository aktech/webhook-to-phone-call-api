# Alerts API

[![Deploy to Fly.io](https://github.com/aktech/alerts-api/actions/workflows/deploy.yml/badge.svg)](https://github.com/aktech/alerts-api/actions/workflows/deploy.yml)

A simple Go service that triggers phone calls via Twilio when a webhook is hit. Uses Twilio's latest API key authentication for production security.

## Features

- GET/POST `/alert/:token` - Triggers phone calls via Twilio (requires matching token)
- **Multiple phone numbers** - Call multiple numbers simultaneously (comma-separated)
- GET `/health` - Health check endpoint
- GET `/twiml` - Returns TwiML XML for the alert message
- Uses Twilio API keys (recommended for production)
- Structured logging with `log/slog`
- Only standard library dependencies

## Prerequisites

- Go 1.25+
- Twilio account with:
  - Account SID
  - API Key and Secret (create at https://console.twilio.com/us1/develop/runtime/api-keys)
  - A Twilio phone number
- Fly.io account (for deployment)

## Setup

### 1. Create Twilio API Key

Twilio recommends using API keys instead of Auth Tokens for production:

1. Go to https://console.twilio.com/us1/develop/runtime/api-keys
2. Click "Create API Key"
3. Give it a name (e.g., "Alerts API")
4. Save the SID and Secret (you won't see the secret again!)

### 2. Local Development

Create a `.env` file (see `.env.example`):

```bash
cp .env.example .env
# Edit .env with your values
```

Run locally:

```bash
# Load environment variables
export $(cat .env | xargs)

# Run the service
go run main.go
```

Test the endpoint:

```bash
# Check health
curl http://localhost:8080/health

# View TwiML
curl http://localhost:8080/twiml

# Trigger an alert (GET or POST both work)
curl http://localhost:8080/alert/your-secret-token

# Note: Twilio requires a publicly accessible URL for the TwiML callback.
# Local testing will fail with "Url is not a valid URL" error.
# The alert endpoint will work properly once deployed to Fly.io.
# For local testing, use ngrok: ngrok http 8080
```

### 3. Deploy to Fly.io (Fully Automated via GitHub Actions)

This project uses GitHub Actions for **100% automated deployment** to Fly.io. Everything is automated - no manual commands needed!

#### Initial Setup (One-time):

1. **Create Fly.io app:**
   ```bash
   fly auth login
   fly apps create alerts-api
   fly auth token  # Copy this token
   ```

2. **Set GitHub Secrets:**

   Go to your GitHub repository → Settings → Secrets and variables → Actions, and add these secrets:

   | Secret Name | Value |
   |-------------|-------|
   | `FLY_API_TOKEN` | Your Fly.io API token (from step 1) |
   | `TOKEN` | Your webhook authentication token |
   | `TWILIO_ACCOUNT_SID` | Your Twilio Account SID |
   | `TWILIO_API_KEY_SID` | Your Twilio API Key SID |
   | `TWILIO_API_KEY_SECRET` | Your Twilio API Key Secret |
   | `TWILIO_FROM_NUMBER` | Your Twilio phone number |
   | `ALERT_TO_NUMBER` | Phone number to call for alerts |

3. **Deploy:**
   ```bash
   git add .
   git commit -m "Initial commit"
   git push origin main
   ```

**That's it!** GitHub Actions will automatically:
- Set all secrets on Fly.io
- Build and test your app
- Deploy to Fly.io

#### Automated Deployment:

- **Push to main** → Automatically sets secrets, builds, tests, and deploys
- **Pull requests** → Automatically runs tests (no deployment)
- **Secrets management** → Automatically synced from GitHub Secrets to Fly.io

Check deployment status in the **Actions** tab of your GitHub repository.

#### Monitoring:

```bash
# Check app status
fly status

# View logs
fly logs

# Open app in browser
fly open
```

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `TOKEN` | Secret token for authenticating webhook calls | Yes |
| `TWILIO_ACCOUNT_SID` | Your Twilio Account SID | Yes |
| `TWILIO_API_KEY_SID` | Your Twilio API Key SID (starts with SK) | Yes |
| `TWILIO_API_KEY_SECRET` | Your Twilio API Key Secret | Yes |
| `TWILIO_FROM_NUMBER` | Your Twilio phone number (E.164 format) | Yes |
| `ALERT_TO_NUMBER` | Phone number(s) to call when alert triggered (comma-separated for multiple: `+1234567890,+0987654321`) | Yes |
| `PORT` | Port to run the service on (default: 8080) | No |

## Usage

Once deployed, trigger an alert (GET or POST both work):

```bash
# Using GET (easy to use in browser)
curl https://your-app.fly.dev/alert/your-secret-token

# Using POST
curl -X POST https://your-app.fly.dev/alert/your-secret-token
```

This will immediately trigger phone calls to all numbers in `ALERT_TO_NUMBER` with the message "Alert triggered".

**Multiple numbers:** Set `ALERT_TO_NUMBER=+1234567890,+0987654321` to call multiple numbers simultaneously. The service will attempt to call all numbers and log any failures.

## Cost Optimization

The app uses Fly.io's cheapest configuration:
- `shared-cpu-1x` with 256MB memory (~$2/month when running)
- Auto-suspend when idle (`min_machines_running = 0`)
- Auto-start on incoming requests

The app will only run (and cost money) when receiving requests, making it very cost-effective for infrequent alerts.

## Security Notes

- Always use API Keys (not Auth Tokens) for production Twilio applications
- Keep your `TOKEN` secret and rotate it periodically
- All secrets should be set via `fly secrets set` (never commit to git)
- The service enforces HTTPS via Fly.io

## License

MIT
