# SMTP2GO Setup Guide

## Getting Your API Key

1. **Sign up/Login**: Go to [SMTP2GO](https://www.smtp2go.com/) and create an account or log in
2. **Navigate to API Keys**: Go to Settings → API Keys
3. **Create New API Key**: Click "Create New API Key"
4. **Copy the Key**: Save the generated API key securely

## Configuration

### Using .env File (Recommended)

Create a `.env` file in your project root:

```bash
# Copy the example file
cp env.example .env

# Edit the .env file with your actual values
nano .env
```

Your `.env` file should contain:

```bash
# Required: Your SMTP2GO API key
SMTP2GO_API_KEY=your_actual_api_key_here

# Optional: Email sender (defaults to "Uptime Monitor <uptime@alexbates.dev>")
SMTP2GO_SENDER=Uptime Monitor <your-email@domain.com>

# Optional: Alert recipient (defaults to "ajbates93@gmail.com")
ALERT_RECIPIENT=alerts@yourdomain.com
```

### Alternative: Direct Environment Variables

You can also set variables directly in your shell:

```bash
export SMTP2GO_API_KEY="your_actual_api_key_here"
export SMTP2GO_SENDER="Uptime Monitor <your-email@domain.com>"
export ALERT_RECIPIENT="alerts@yourdomain.com"
```

## Running the Application

The application automatically loads `.env` files using the `godotenv` package.

### Using Makefile

```bash
# Run directly with go run
make run

# Build and run
make run-api
```

### Manual Execution

```bash
# Run directly (loads .env automatically)
go run ./cmd/api

# Build and run
go build ./cmd/api
./api
```

## API vs SMTP

### API Method (Recommended)
- **URL**: `https://api.smtp2go.com/v3/email/send`
- **Method**: POST
- **Authentication**: API Key in request body
- **Advantages**: 
  - More secure
  - Better rate limiting
  - Detailed delivery reports
  - No connection management needed

### SMTP Method (Alternative)
If you prefer SMTP, you can use:
- **Host**: `mail.smtp2go.com`
- **Port**: `2525` or `587`
- **Username**: Your SMTP2GO username
- **Password**: Your SMTP2GO password

## Testing

1. **Set your environment variables** with your actual API key
2. **Run the application**: `make run` or `make run-api`
3. **Test alerts**: The system will automatically send alerts when websites go down/up

## Security Notes

- ✅ **Environment variables** keep sensitive data out of your code
- ✅ **Never commit** your `.env` file to version control
- ✅ **Add `.env` to `.gitignore`** to prevent accidental commits
- ✅ **API keys can be revoked** and regenerated if compromised
- ✅ **Use different keys** for development and production

## Rate Limits

SMTP2GO has generous rate limits:
- Free tier: 1,000 emails/month
- Paid tiers: Higher limits available

The alert system is designed to prevent spam by:
- Sending "down" alerts only once per hour per website
- Sending "recovery" alerts only once per 24 hours per website 