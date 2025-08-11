#!/bin/bash

# AWS Infrastructure Deployment Script for Master Ovasabi
# This script sets up the complete AWS infrastructure automatically

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_info() {
    echo -e "${BLUE}[ℹ]${NC} $1"
}

print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Configuration
STACK_NAME="master-ovasabi"
AWS_REGION="${AWS_REGION:-af-south-1}"
AWS_ACCOUNT_ID="${AWS_ACCOUNT_ID:-322424815667}"

# Check prerequisites
check_prerequisites() {
    print_step "Checking prerequisites..."
    
    # Check AWS CLI
    if ! command -v aws &> /dev/null; then
        print_error "AWS CLI is not installed. Please install it first."
        echo "Visit: https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html"
        exit 1
    fi
    
    # Check AWS credentials
    if ! aws sts get-caller-identity &> /dev/null; then
        print_error "AWS credentials are not configured. Please run 'aws configure'."
        exit 1
    fi
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install it first."
        exit 1
    fi
    
    # Check environment variables
    if [ -f ".env" ]; then
        source .env
    else
        print_error ".env file not found. Please create one from .env.example"
        exit 1
    fi
    
    print_status "Prerequisites check passed"
}

# Generate secure passwords
generate_passwords() {
    print_step "Generating secure passwords..."
    
    DB_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
    REDIS_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
    
    print_status "Passwords generated securely"
}

# Deploy infrastructure using CloudFormation
deploy_infrastructure() {
    print_step "Deploying AWS infrastructure..."
    
    # Check if stack already exists
    if aws cloudformation describe-stacks --stack-name "${STACK_NAME}-infrastructure" --region "$AWS_REGION" &> /dev/null; then
        print_info "Infrastructure stack already exists. Updating..."
        aws cloudformation deploy \
            --template-file deployments/aws/infrastructure.yaml \
            --stack-name "${STACK_NAME}-infrastructure" \
            --parameter-overrides \
                Environment=production \
                DBPassword="$DB_PASSWORD" \
                RedisPassword="$REDIS_PASSWORD" \
                AdminPassword="$ADMIN_PASSWORD" \
            --capabilities CAPABILITY_IAM \
            --region "$AWS_REGION"
    else
        print_info "Creating new infrastructure stack..."
        aws cloudformation deploy \
            --template-file deployments/aws/infrastructure.yaml \
            --stack-name "${STACK_NAME}-infrastructure" \
            --parameter-overrides \
                Environment=production \
                DBPassword="$DB_PASSWORD" \
                RedisPassword="$REDIS_PASSWORD" \
                AdminPassword="$ADMIN_PASSWORD" \
            --capabilities CAPABILITY_IAM \
            --region "$AWS_REGION"
    fi
    
    print_status "Infrastructure deployment completed"
}

# Build and push Docker images
build_and_push_images() {
    print_step "Building and pushing Docker images to ECR..."
    
    # Run the ECR deployment script
    if [ -x "./deployments/aws/deploy-ecr.sh" ]; then
        ./deployments/aws/deploy-ecr.sh
    else
        print_error "ECR deployment script not found or not executable"
        exit 1
    fi
    
    print_status "Images pushed to ECR successfully"
}

# Get infrastructure outputs
get_infrastructure_outputs() {
    print_step "Retrieving infrastructure information..."
    
    # Get stack outputs
    VPC_ID=$(aws cloudformation describe-stacks \
        --stack-name "${STACK_NAME}-infrastructure" \
        --query 'Stacks[0].Outputs[?OutputKey==`VPCId`].OutputValue' \
        --output text --region "$AWS_REGION")
    
    DB_ENDPOINT=$(aws cloudformation describe-stacks \
        --stack-name "${STACK_NAME}-infrastructure" \
        --query 'Stacks[0].Outputs[?OutputKey==`DatabaseEndpoint`].OutputValue' \
        --output text --region "$AWS_REGION")
    
    REDIS_ENDPOINT=$(aws cloudformation describe-stacks \
        --stack-name "${STACK_NAME}-infrastructure" \
        --query 'Stacks[0].Outputs[?OutputKey==`RedisEndpoint`].OutputValue' \
        --output text --region "$AWS_REGION")
    
    CLUSTER_NAME=$(aws cloudformation describe-stacks \
        --stack-name "${STACK_NAME}-infrastructure" \
        --query 'Stacks[0].Outputs[?OutputKey==`ECSClusterName`].OutputValue' \
        --output text --region "$AWS_REGION")
    
    ALB_DNS=$(aws cloudformation describe-stacks \
        --stack-name "${STACK_NAME}-infrastructure" \
        --query 'Stacks[0].Outputs[?OutputKey==`LoadBalancerDNS`].OutputValue' \
        --output text --region "$AWS_REGION")
    
    print_status "Infrastructure information retrieved"
}

# Create ECS task definition
create_task_definition() {
    print_step "Creating ECS task definition..."
    
    cat > task-definition-prod.json << EOF
{
  "family": "${STACK_NAME}",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "1024",
  "memory": "2048",
  "executionRoleArn": "arn:aws:iam::${AWS_ACCOUNT_ID}:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::${AWS_ACCOUNT_ID}:role/ecsTaskRole",
  "containerDefinitions": [
    {
      "name": "app",
      "image": "${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/master-ovasabi:latest",
      "essential": true,
      "portMappings": [
        {"containerPort": 8080, "protocol": "tcp"},
        {"containerPort": 8081, "protocol": "tcp"}
      ],
      "environment": [
        {"name": "APP_ENV", "value": "production"},
        {"name": "APP_NAME", "value": "master-ovasabi"},
        {"name": "DB_HOST", "value": "${DB_ENDPOINT}"},
        {"name": "DB_PORT", "value": "5432"},
        {"name": "DB_NAME", "value": "master_ovasabi"},
        {"name": "REDIS_HOST", "value": "${REDIS_ENDPOINT}"},
        {"name": "REDIS_PORT", "value": "6379"},
        {"name": "HTTP_PORT", "value": "8081"},
        {"name": "GRPC_PORT", "value": "8082"},
        {"name": "LOG_LEVEL", "value": "info"}
      ],
      "secrets": [
        {"name": "DB_PASSWORD", "valueFrom": "arn:aws:secretsmanager:${AWS_REGION}:${AWS_ACCOUNT_ID}:secret:${STACK_NAME}-infrastructure-db-password"},
        {"name": "REDIS_PASSWORD", "valueFrom": "arn:aws:secretsmanager:${AWS_REGION}:${AWS_ACCOUNT_ID}:secret:${STACK_NAME}-infrastructure-redis-password"},
        {"name": "ADMIN_PASSWORD", "valueFrom": "arn:aws:secretsmanager:${AWS_REGION}:${AWS_ACCOUNT_ID}:secret:${STACK_NAME}-infrastructure-admin-password"}
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/${STACK_NAME}",
          "awslogs-region": "${AWS_REGION}",
          "awslogs-stream-prefix": "app"
        }
      },
      "healthCheck": {
        "command": ["CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"],
        "interval": 30,
        "timeout": 5,
        "retries": 3,
        "startPeriod": 60
      }
    }
  ]
}
EOF
    
    # Register task definition
    aws ecs register-task-definition \
        --cli-input-json file://task-definition-prod.json \
        --region "$AWS_REGION" > /dev/null
    
    # Clean up
    rm task-definition-prod.json
    
    print_status "Task definition created"
}

# Deploy ECS service
deploy_ecs_service() {
    print_step "Deploying ECS service..."
    
    # Get networking information
    SUBNET_IDS=$(aws ec2 describe-subnets \
        --filters "Name=vpc-id,Values=$VPC_ID" "Name=tag:Name,Values=*public*" \
        --query 'Subnets[].SubnetId' \
        --output text --region "$AWS_REGION" | tr '\t' ',')
    
    SECURITY_GROUP_ID=$(aws ec2 describe-security-groups \
        --filters "Name=vpc-id,Values=$VPC_ID" "Name=group-name,Values=*ECS*" \
        --query 'SecurityGroups[0].GroupId' \
        --output text --region "$AWS_REGION")
    
    TARGET_GROUP_ARN=$(aws elbv2 describe-target-groups \
        --names "${STACK_NAME}-infrastructure-tg" \
        --query 'TargetGroups[0].TargetGroupArn' \
        --output text --region "$AWS_REGION")
    
    # Check if service exists
    if aws ecs describe-services --cluster "$CLUSTER_NAME" --services "${STACK_NAME}-service" --region "$AWS_REGION" &> /dev/null; then
        print_info "Updating existing ECS service..."
        aws ecs update-service \
            --cluster "$CLUSTER_NAME" \
            --service "${STACK_NAME}-service" \
            --task-definition "$STACK_NAME" \
            --desired-count 1 \
            --region "$AWS_REGION" > /dev/null
    else
        print_info "Creating new ECS service..."
        aws ecs create-service \
            --cluster "$CLUSTER_NAME" \
            --service-name "${STACK_NAME}-service" \
            --task-definition "$STACK_NAME" \
            --desired-count 1 \
            --launch-type FARGATE \
            --network-configuration "awsvpcConfiguration={subnets=[$SUBNET_IDS],securityGroups=[$SECURITY_GROUP_ID],assignPublicIp=ENABLED}" \
            --load-balancers "targetGroupArn=$TARGET_GROUP_ARN,containerName=app,containerPort=8080" \
            --region "$AWS_REGION" > /dev/null
    fi
    
    print_status "ECS service deployed"
}

# Wait for service to be stable
wait_for_service() {
    print_step "Waiting for service to become stable..."
    
    print_info "This may take a few minutes..."
    aws ecs wait services-stable \
        --cluster "$CLUSTER_NAME" \
        --services "${STACK_NAME}-service" \
        --region "$AWS_REGION"
    
    print_status "Service is stable and running"
}

# Display deployment information
show_deployment_info() {
    print_step "Deployment completed successfully! 🎉"
    
    echo
    echo "=== Deployment Information ==="
    echo "Application URL: http://${ALB_DNS}"
    echo "AWS Region: ${AWS_REGION}"
    echo "Stack Name: ${STACK_NAME}-infrastructure"
    echo "ECS Cluster: ${CLUSTER_NAME}"
    echo "Database Endpoint: ${DB_ENDPOINT}"
    echo "Redis Endpoint: ${REDIS_ENDPOINT}"
    echo
    echo "=== Service Management ==="
    echo "View service status:"
    echo "  aws ecs describe-services --cluster ${CLUSTER_NAME} --services ${STACK_NAME}-service --region ${AWS_REGION}"
    echo
    echo "View logs:"
    echo "  aws logs tail /ecs/${STACK_NAME} --follow --region ${AWS_REGION}"
    echo
    echo "Scale service:"
    echo "  aws ecs update-service --cluster ${CLUSTER_NAME} --service ${STACK_NAME}-service --desired-count 2 --region ${AWS_REGION}"
    echo
    echo "=== Monitoring ==="
    echo "CloudWatch Logs: https://console.aws.amazon.com/cloudwatch/home?region=${AWS_REGION}#logsV2:log-groups/log-group/\$252Fecs\$252F${STACK_NAME}"
    echo "ECS Console: https://console.aws.amazon.com/ecs/home?region=${AWS_REGION}#/clusters/${CLUSTER_NAME}/services"
    echo
}

# Cleanup function
cleanup() {
    print_info "Cleaning up temporary files..."
    rm -f task-definition-prod.json
}

# Main deployment function
main() {
    echo "=== Master Ovasabi AWS Deployment ==="
    echo "This script will deploy your application to AWS ECS with Fargate"
    echo
    
    # Set trap for cleanup
    trap cleanup EXIT
    
    check_prerequisites
    generate_passwords
    deploy_infrastructure
    build_and_push_images
    get_infrastructure_outputs
    create_task_definition
    deploy_ecs_service
    wait_for_service
    show_deployment_info
    
    print_status "Deployment completed successfully!"
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
