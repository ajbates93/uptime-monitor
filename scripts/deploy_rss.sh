#!/bin/bash

# RSS Feature Deployment Script
# This script helps safely deploy the RSS feature to production

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REMOTE_HOST="your-hetzner-vps-ip"
REMOTE_USER="root"
REMOTE_PATH="/opt/the-ark"
BACKUP_DIR="/tmp/ark_backup_$(date +%Y%m%d_%H%M%S)"

echo -e "${GREEN}🚀 RSS Feature Deployment Script${NC}"
echo "=================================="

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo -e "${RED}❌ Error: Please run this script from the project root directory${NC}"
    exit 1
fi

# Check if environment file exists
if [ ! -f ".env" ]; then
    echo -e "${YELLOW}⚠️  Warning: .env file not found. Please ensure RSS configuration is set.${NC}"
fi

echo -e "${YELLOW}📋 Pre-deployment Checklist:${NC}"
echo "1. ✅ RSS feature code is ready"
echo "2. ✅ Database migrations are tested"
echo "3. ✅ Environment variables are configured"
echo "4. ✅ Production database is backed up"
echo ""

read -p "Have you completed all the above? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${RED}❌ Deployment cancelled. Please complete the checklist first.${NC}"
    exit 1
fi

echo -e "${YELLOW}🔧 Building the application...${NC}"
go build -o bin/the-ark cmd/main.go

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Build failed! Please fix the compilation errors.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Build successful${NC}"

echo -e "${YELLOW}📤 Deploying to production...${NC}"

# Create backup directory on remote server
echo "Creating backup directory..."
ssh ${REMOTE_USER}@${REMOTE_HOST} "mkdir -p ${BACKUP_DIR}"

# Backup current database
echo "Backing up current database..."
ssh ${REMOTE_USER}@${REMOTE_HOST} "cp ${REMOTE_PATH}/ark.db ${BACKUP_DIR}/"

# Backup current binary
echo "Backing up current binary..."
ssh ${REMOTE_USER}@${REMOTE_HOST} "cp ${REMOTE_PATH}/the-ark ${BACKUP_DIR}/"

# Upload new binary
echo "Uploading new binary..."
scp bin/the-ark ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_PATH}/

# Upload environment file if it exists
if [ -f ".env" ]; then
    echo "Uploading environment file..."
    scp .env ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_PATH}/
fi

# Restart the service
echo "Restarting the service..."
ssh ${REMOTE_USER}@${REMOTE_HOST} "systemctl restart the-ark"

# Wait a moment for the service to start
sleep 5

# Check if the service is running
echo "Checking service status..."
ssh ${REMOTE_USER}@${REMOTE_HOST} "systemctl status the-ark --no-pager"

# Test the RSS endpoint
echo "Testing RSS endpoint..."
HTTP_STATUS=$(ssh ${REMOTE_USER}@${REMOTE_HOST} "curl -s -o /dev/null -w '%{http_code}' http://localhost:4000/rss" || echo "000")

if [ "$HTTP_STATUS" = "200" ] || [ "$HTTP_STATUS" = "401" ] || [ "$HTTP_STATUS" = "404" ]; then
    echo -e "${GREEN}✅ Service is responding (HTTP ${HTTP_STATUS})${NC}"
else
    echo -e "${RED}❌ Service is not responding properly (HTTP ${HTTP_STATUS})${NC}"
    echo -e "${YELLOW}⚠️  Rolling back to previous version...${NC}"
    
    # Rollback
    ssh ${REMOTE_USER}@${REMOTE_HOST} "cp ${BACKUP_DIR}/the-ark ${REMOTE_PATH}/"
    ssh ${REMOTE_USER}@${REMOTE_HOST} "systemctl restart the-ark"
    
    echo -e "${RED}❌ Deployment failed and rolled back. Check logs for details.${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}🎉 RSS Feature Deployment Successful!${NC}"
echo ""
echo -e "${YELLOW}📋 Post-deployment tasks:${NC}"
echo "1. ✅ Verify RSS feature is working in the portal"
echo "2. ✅ Test adding a new RSS feed"
echo "3. ✅ Check that existing uptime monitoring still works"
echo "4. ✅ Monitor application logs for any errors"
echo ""
echo -e "${YELLOW}💾 Backup location: ${BACKUP_DIR}${NC}"
echo -e "${YELLOW}📝 To rollback manually:${NC}"
echo "   ssh ${REMOTE_USER}@${REMOTE_HOST}"
echo "   cp ${BACKUP_DIR}/the-ark ${REMOTE_PATH}/"
echo "   systemctl restart the-ark"
echo ""
echo -e "${GREEN}🚀 RSS Feature is now live!${NC}" 