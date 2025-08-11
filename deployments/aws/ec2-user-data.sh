#!/bin/bash

# EC2 User Data Script for Master Ovasabi
# This script installs Docker and sets up the application

set -e

# Update system
yum update -y

# Install Docker
yum install -y docker
systemctl start docker
systemctl enable docker
usermod -a -G docker ec2-user

# Install Docker Compose
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose

# Create application directory
mkdir -p /opt/master-ovasabi
cd /opt/master-ovasabi

# Clone repository (you'll need to update this with your actual repo)
# git clone https://github.com/nmxmxh/master-ovasabi.git .

# Create .env file (you'll need to customize this)
cat > .env << 'ENV_EOF'
APP_ENV=production
APP_NAME=master-ovasabi
HTTP_PORT=8081
GRPC_PORT=8082

POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_secure_postgres_password
POSTGRES_DB=master_ovasabi
DB_USER=postgres
DB_PASSWORD=your_secure_postgres_password
DB_NAME=master_ovasabi
DB_PORT=5432

REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=your_secure_redis_password

ADMIN_USER=nmxmxh
ADMIN_PASSWORD=your_secure_admin_password

CAMPAIGN_ID=0
ENV_EOF

# Note: You would need to copy your application files here
# For now, this is just a template

echo "EC2 setup complete. You need to:"
echo "1. Copy your application code to /opt/master-ovasabi"
echo "2. Update the .env file with your actual values"
echo "3. Run: docker-compose up -d"
