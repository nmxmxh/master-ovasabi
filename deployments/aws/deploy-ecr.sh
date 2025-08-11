#!/bin/bash

# AWS ECR Deployment Script for Master Ovasabi
# This script builds, tags, and pushes Docker images to AWS ECR

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Check if required environment variables are set
check_env_vars() {
    print_step "Checking environment variables..."
    
    if [ -z "$AWS_REGION" ]; then
        print_error "AWS_REGION is not set. Please set it in your .env file or environment."
        exit 1
    fi
    
    if [ -z "$AWS_ACCOUNT_ID" ]; then
        print_error "AWS_ACCOUNT_ID is not set. Please set it in your .env file or environment."
        exit 1
    fi
    
    if [ -z "$ECR_REGISTRY" ]; then
        export ECR_REGISTRY="$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"
        print_status "ECR_REGISTRY set to: $ECR_REGISTRY"
    fi
    
    print_status "Environment variables check passed."
}

# Check if AWS CLI is installed and configured
check_aws_cli() {
    print_step "Checking AWS CLI..."
    
    if ! command -v aws &> /dev/null; then
        print_error "AWS CLI is not installed. Please install it first."
        exit 1
    fi
    
    # Check AWS credentials
    if ! aws sts get-caller-identity &> /dev/null; then
        print_error "AWS credentials are not configured. Please run 'aws configure'."
        exit 1
    fi
    
    local account_id=$(aws sts get-caller-identity --query Account --output text)
    if [ "$account_id" != "$AWS_ACCOUNT_ID" ]; then
        print_warning "AWS_ACCOUNT_ID in .env ($AWS_ACCOUNT_ID) doesn't match current AWS account ($account_id)"
        export AWS_ACCOUNT_ID="$account_id"
        export ECR_REGISTRY="$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"
        print_status "Updated AWS_ACCOUNT_ID to: $AWS_ACCOUNT_ID"
    fi
    
    print_status "AWS CLI check passed."
}

# Create ECR repositories if they don't exist
create_ecr_repositories() {
    print_step "Creating ECR repositories..."
    
    local repositories=("master-ovasabi" "nexus" "media-streaming" "ws-gateway" "nginx")
    
    for repo in "${repositories[@]}"; do
        if ! aws ecr describe-repositories --repository-names "$repo" --region "$AWS_REGION" &> /dev/null; then
            print_status "Creating ECR repository: $repo"
            aws ecr create-repository \
                --repository-name "$repo" \
                --image-scanning-configuration scanOnPush=true \
                --region "$AWS_REGION" &> /dev/null
        else
            print_status "ECR repository already exists: $repo"
        fi
    done
    
    print_status "ECR repositories ready."
}

# Login to ECR
ecr_login() {
    print_step "Logging into ECR..."
    
    aws ecr get-login-password --region "$AWS_REGION" | \
        docker login --username AWS --password-stdin "$ECR_REGISTRY"
    
    print_status "ECR login successful."
}

# Build images
build_images() {
    print_step "Building Docker images..."
    
    # Build the common builder image first
    print_status "Building common builder image..."
    if ! docker build -f deployments/docker/Dockerfile.builder -t ovasabi-go-builder:latest .; then
        print_error "Failed to build common builder image"
        exit 1
    fi
    
    # Use docker-compose to build images with ECR tags
    print_status "Building application images..."
    docker compose -f deployments/docker/docker-compose.prod.yml build
    
    print_status "All images built successfully."
}

# Push images to ECR
push_images() {
    print_step "Pushing images to ECR..."
    
    docker compose -f deployments/docker/docker-compose.prod.yml push
    
    print_status "All images pushed successfully."
}

# Generate deployment manifest
generate_manifest() {
    print_step "Generating deployment manifest..."
    
    cat > deployments/aws/deployment-manifest.yaml << EOF
# Deployment Manifest for Master Ovasabi
# Generated on: $(date)

AWS_REGION: $AWS_REGION
AWS_ACCOUNT_ID: $AWS_ACCOUNT_ID
ECR_REGISTRY: $ECR_REGISTRY

Images:
  app: $ECR_REGISTRY/master-ovasabi:latest
  nexus: $ECR_REGISTRY/nexus:latest
  media-streaming: $ECR_REGISTRY/media-streaming:latest
  ws-gateway: $ECR_REGISTRY/ws-gateway:latest
  nginx: $ECR_REGISTRY/nginx:latest

# Next Steps:
# 1. Deploy infrastructure using CloudFormation or Terraform
# 2. Create ECS task definitions with the above image URIs
# 3. Deploy services to ECS
# 4. Set up Application Load Balancer
# 5. Configure Route 53 for custom domain (optional)
EOF
    
    print_status "Deployment manifest created at: deployments/aws/deployment-manifest.yaml"
}

# Display next steps
show_next_steps() {
    print_step "Deployment preparation complete!"
    
    echo
    echo "=== Docker Images Ready ==="
    echo "✅ Images built and pushed to ECR"
    echo "✅ Registry: $ECR_REGISTRY"
    echo
    echo "=== Next Steps for AWS Deployment ==="
    echo "1. 🏗️  Deploy Infrastructure:"
    echo "   - Use AWS CloudFormation, CDK, or Terraform"
    echo "   - Create VPC, subnets, security groups"
    echo "   - Set up RDS (PostgreSQL) and ElastiCache (Redis)"
    echo "   - Create ECS cluster"
    echo
    echo "2. 🚀 Deploy Application:"
    echo "   - Create ECS task definitions using the images above"
    echo "   - Deploy ECS services"
    echo "   - Set up Application Load Balancer"
    echo "   - Configure target groups and health checks"
    echo
    echo "3. 🔐 Security & Configuration:"
    echo "   - Store secrets in AWS Secrets Manager"
    echo "   - Update environment variables for production"
    echo "   - Configure IAM roles and policies"
    echo
    echo "4. 🌐 Networking (Optional):"
    echo "   - Set up custom domain with Route 53"
    echo "   - Configure SSL certificates with ACM"
    echo "   - Set up CloudFront CDN if needed"
    echo
    echo "=== Available Images ==="
    echo "App:            $ECR_REGISTRY/master-ovasabi:latest"
    echo "Nexus:          $ECR_REGISTRY/nexus:latest"
    echo "Media Streaming: $ECR_REGISTRY/media-streaming:latest"
    echo "WS Gateway:     $ECR_REGISTRY/ws-gateway:latest"
    echo "Nginx:          $ECR_REGISTRY/nginx:latest"
    echo
    echo "Check deployments/aws/deployment-manifest.yaml for details."
}

# Main function
main() {
    echo "=== AWS ECR Deployment Script ==="
    echo "Building and pushing Master Ovasabi to AWS ECR"
    echo
    
    # Load environment variables if .env exists
    if [ -f ".env" ]; then
        print_status "Loading environment variables from .env"
        set -a
        source .env
        set +a
    else
        print_warning "No .env file found. Please create one based on .env.example"
        print_warning "Continuing with environment variables from shell..."
    fi
    
    check_env_vars
    check_aws_cli
    create_ecr_repositories
    ecr_login
    build_images
    push_images
    generate_manifest
    show_next_steps
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
