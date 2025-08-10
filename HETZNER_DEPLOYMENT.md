# Hetzner VPS Deployment Guide - The Ark Uptime Monitor

This guide is specifically tailored for deploying on a blank Hetzner VPS with minimal resources.

## Prerequisites

- Hetzner VPS (1GB RAM minimum, 2GB recommended)
- Ubuntu 20.04+ or Debian 11+
- SSH access to your server

## Step 1: Initial Server Setup

### Connect to your VPS
```bash
ssh root@your-server-ip
```

### Update system and install essentials
```bash
# Update package lists
apt update && apt upgrade -y

# Install essential packages
apt install -y curl wget git sqlite3 nano htop ufw

# Install Go (if not already installed)
if ! command -v go &> /dev/null; then
    echo "Installing Go..."
    wget https://go.dev/dl/go1.24.5.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.24.5.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    source ~/.bashrc
    rm go1.24.5.linux-amd64.tar.gz
fi

# Verify Go installation
go version
```

### Basic security setup
```bash
# Configure firewall
ufw allow ssh
ufw allow 80
ufw allow 443
ufw --force enable

# Create non-root user (optional but recommended)
adduser theark
usermod -aG sudo theark
```

## Step 2: Build and Deploy

### Option A: Build on your local machine and transfer

1. **On your local machine:**
   ```bash
   # Build for Linux
   GOOS=linux GOARCH=amd64 make build
   
   # Create deployment package
   ./deploy.sh
   ```

2. **Transfer to server:**
   ```bash
   scp -r deploy/* root@your-server-ip:/tmp/the-ark/
   ```

### Option B: Build directly on the server

1. **Clone your repository:**
   ```bash
   git clone https://github.com/your-username/uptime-monitor.git
   cd uptime-monitor
   ```

2. **Build on server:**
   ```bash
   make build
   make generate
   ```

## Step 3: Install and Configure

### Create directories and set permissions
```bash
# Create application directories
mkdir -p /opt/the-ark
mkdir -p /var/lib/the-ark

# Set ownership (using nobody for simplicity)
chown nobody:nogroup /opt/the-ark
chown nobody:nogroup /var/lib/the-ark
```

### Copy files
```bash
# If you transferred files
cp /tmp/the-ark/the-ark /opt/the-ark/
cp /tmp/the-ark/the-ark.service /etc/systemd/system/

# Or if you built on server
cp bin/the-ark /opt/the-ark/
cp the-ark.service /etc/systemd/system/

# Set permissions
chmod +x /opt/the-ark/the-ark
```

### Configure environment
```bash
# Copy and edit environment file
cp deployment.env.example /opt/the-ark/.env
nano /opt/the-ark/.env
```

**Required environment variables to set:**
```bash
ARK_SESSION_SECRET=your-super-secure-random-string-here
ARK_ADMIN_EMAIL=your-email@domain.com
ARK_ADMIN_PASSWORD=your-secure-password
ARK_SMTP2GO_API_KEY=your_smtp2go_api_key
```

### Set file permissions
```bash
chown nobody:nogroup /opt/the-ark/.env
```

## Step 4: Start the Service

```bash
# Reload systemd and start service
systemctl daemon-reload
systemctl enable the-ark
systemctl start the-ark

# Check status
systemctl status the-ark
journalctl -u the-ark -f
```

## Step 5: Reverse Proxy Setup (Optional but Recommended)

### Install Nginx
```bash
apt install -y nginx
```

### Create Nginx configuration
```bash
nano /etc/nginx/sites-available/the-ark
```

Add this content:
```nginx
server {
    listen 80;
    server_name your-domain.com;  # Replace with your domain

    location / {
        proxy_pass http://localhost:4000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Enable the site
```bash
ln -s /etc/nginx/sites-available/the-ark /etc/nginx/sites-enabled/
nginx -t
systemctl reload nginx
```

## Step 6: SSL Setup (Optional)

### Install Certbot
```bash
apt install -y certbot python3-certbot-nginx
```

### Get SSL certificate
```bash
certbot --nginx -d your-domain.com
```

## Resource Monitoring

### Check resource usage
```bash
# Memory usage
free -h

# Disk usage
df -h

# Process monitoring
htop

# Application logs
journalctl -u the-ark -f
```

### Expected resource usage:
- **Memory**: 10-50MB RAM
- **CPU**: Minimal (only during site checks)
- **Disk**: ~50MB (binary + database)
- **Network**: Minimal (only HTTP requests to monitored sites)

## Troubleshooting

### Service won't start
```bash
# Check logs
journalctl -u the-ark -n 50

# Check permissions
ls -la /opt/the-ark/
ls -la /var/lib/the-ark/

# Check environment file
cat /opt/the-ark/.env
```

### Permission issues
```bash
# Fix ownership
chown -R nobody:nogroup /opt/the-ark
chown -R nobody:nogroup /var/lib/the-ark
```

### Database issues
```bash
# Check database file
ls -la /var/lib/the-ark/ark.db

# Check disk space
df -h
```

## Maintenance

### Update the application
```bash
# Stop service
systemctl stop the-ark

# Backup database
cp /var/lib/the-ark/ark.db /var/lib/the-ark/ark.db.backup

# Copy new binary
cp new-the-ark /opt/the-ark/the-ark
chmod +x /opt/the-ark/the-ark

# Start service
systemctl start the-ark
```

### Backup strategy
```bash
# Create backup script
nano /opt/backup-the-ark.sh
```

Add this content:
```bash
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
cp /var/lib/the-ark/ark.db /backup/ark_$DATE.db
# Keep only last 7 days of backups
find /backup -name "ark_*.db" -mtime +7 -delete
```

Make it executable:
```bash
chmod +x /opt/backup-the-ark.sh
```

Add to crontab:
```bash
crontab -e
# Add: 0 2 * * * /opt/backup-the-ark.sh
```

## Security Notes

1. **Change default SSH port** (optional)
2. **Use SSH keys instead of passwords**
3. **Keep system updated regularly**
4. **Monitor logs for suspicious activity**
5. **Use strong passwords for admin account**
6. **Regular backups**

This setup should work perfectly on even the smallest Hetzner VPS while using minimal resources. 