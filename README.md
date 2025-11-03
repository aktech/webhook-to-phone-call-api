# Alerts API

A simple Go service that triggers phone calls via Twilio when a webhook is hit. Uses Twilio's latest API key authentication for production security.

## Features

- POST `/alert/:token` - Triggers a phone call via Twilio (requires matching token)
- GET `/health` - Health check endpoint
- GET `/twiml` - Returns TwiML XML for the alert message
- Uses Twilio API keys (recommended for production)
- Structured logging with `log/slog`
- Only standard library dependencies

## Prerequisites

- Go 1.23+
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

### 3. Deploy to Fly.io

```bash
# Login to Fly.io
fly auth login

# Launch the app (follow prompts)
fly launch

# Set environment variables (secrets)
fly secrets set TOKEN=your-secret-token-here
fly secrets set TWILIO_ACCOUNT_SID=ACxxxxx
fly secrets set TWILIO_API_KEY_SID=SKxxxxx
fly secrets set TWILIO_API_KEY_SECRET=your-api-key-secret
fly secrets set TWILIO_FROM_NUMBER=+1234567890
fly secrets set ALERT_TO_NUMBER=+1234567890

# Deploy
fly deploy

# Check status
fly status

# View logs
fly logs
```

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `TOKEN` | Secret token for authenticating webhook calls | Yes |
| `TWILIO_ACCOUNT_SID` | Your Twilio Account SID | Yes |
| `TWILIO_API_KEY_SID` | Your Twilio API Key SID (starts with SK) | Yes |
| `TWILIO_API_KEY_SECRET` | Your Twilio API Key Secret | Yes |
| `TWILIO_FROM_NUMBER` | Your Twilio phone number (E.164 format) | Yes |
| `ALERT_TO_NUMBER` | Phone number to call when alert triggered | Yes |
| `PORT` | Port to run the service on (default: 8080) | No |

## Usage

Once deployed, trigger an alert (GET or POST both work):

```bash
# Using GET (easy to use in browser)
curl https://your-app.fly.dev/alert/your-secret-token

# Using POST
curl -X POST https://your-app.fly.dev/alert/your-secret-token
```

This will immediately trigger a phone call to `ALERT_TO_NUMBER` with the message "Alert triggered".

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
