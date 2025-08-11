# AWS Infrastructure Setup Guide for Master Ovasabi

This guide walks you through setting up the complete AWS infrastructure for Master Ovasabi deployment.

## 🎯 Overview

We'll set up a production-ready infrastructure with:

- **ECS Fargate** for container orchestration
- **RDS PostgreSQL** for database
- **ElastiCache Redis** for caching
- **Application Load Balancer** for traffic distribution
- **ECR** for container registry
- **VPC** with public/private subnets
- **Secrets Manager** for sensitive data

## 📋 Prerequisites

### 1. AWS Account Setup

- AWS Account with admin permissions
- AWS CLI installed and configured
- Docker installed locally

### 2. Required AWS Services Permissions

Ensure your AWS user/role has permissions for:

- EC2, VPC, ECS, ECR
- RDS, ElastiCache
- IAM, Secrets Manager
- CloudFormation
- Application Load Balancer

## 🚀 Step-by-Step Deployment

### Step 1: Configure AWS CLI

```bash
# Install AWS CLI if not already installed
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install

# Configure AWS credentials
aws configure
# Enter your:
# - AWS Access Key ID
# - AWS Secret Access Key  
# - Default region (e.g., af-south-1)
# - Default output format (json)

# Verify configuration
aws sts get-caller-identity
```

### Step 2: Prepare Your Environment

```bash
# Navigate to your project
cd /path/to/master-ovasabi

# Ensure your .env is configured with AWS settings
# Edit .env and set:
# AWS_REGION=af-south-1
# AWS_ACCOUNT_ID=322424815667
# ECR_REGISTRY=322424815667.dkr.ecr.af-south-1.amazonaws.com

# Validate environment
make validate-env
```

### Step 3: Deploy Infrastructure (CloudFormation)

```bash
# Create the infrastructure stack
aws cloudformation deploy \
  --template-file deployments/aws/infrastructure.yaml \
  --stack-name master-ovasabi-infrastructure \
  --parameter-overrides \
    Environment=production \
    DBPassword="$(openssl rand -base64 32)" \
    RedisPassword="$(openssl rand -base64 32)" \
    AdminPassword="your-secure-admin-password" \
  --capabilities CAPABILITY_IAM \
  --region af-south-1

# Wait for stack creation (10-15 minutes)
aws cloudformation wait stack-create-complete \
  --stack-name master-ovasabi-infrastructure \
  --region af-south-1

# Get stack outputs
aws cloudformation describe-stacks \
  --stack-name master-ovasabi-infrastructure \
  --query 'Stacks[0].Outputs' \
  --region af-south-1
```

### Step 4: Build and Push Container Images

```bash
# Build and push all images to ECR
make aws-ecr-deploy

# Or manually:
./deployments/aws/deploy-ecr.sh
```

### Step 5: Create ECS Task Definition

```bash
# Get infrastructure outputs
VPC_ID=$(aws cloudformation describe-stacks \
  --stack-name master-ovasabi-infrastructure \
  --query 'Stacks[0].Outputs[?OutputKey==`VPCId`].OutputValue' \
  --output text --region af-south-1)

DB_ENDPOINT=$(aws cloudformation describe-stacks \
  --stack-name master-ovasabi-infrastructure \
  --query 'Stacks[0].Outputs[?OutputKey==`DatabaseEndpoint`].OutputValue' \
  --output text --region af-south-1)

REDIS_ENDPOINT=$(aws cloudformation describe-stacks \
  --stack-name master-ovasabi-infrastructure \
  --query 'Stacks[0].Outputs[?OutputKey==`RedisEndpoint`].OutputValue' \
  --output text --region af-south-1)

# Create a production task definition
cat > task-definition-prod.json << EOF
{
  "family": "master-ovasabi-prod",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "2048",
  "memory": "4096",
  "executionRoleArn": "arn:aws:iam::322424815667:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::322424815667:role/ecsTaskRole",
  "containerDefinitions": [
    {
      "name": "app",
      "image": "322424815667.dkr.ecr.af-south-1.amazonaws.com/master-ovasabi:latest",
      "essential": true,
      "portMappings": [
        {"containerPort": 8080, "protocol": "tcp"},
        {"containerPort": 8081, "protocol": "tcp"}
      ],
      "environment": [
        {"name": "APP_ENV", "value": "production"},
        {"name": "DB_HOST", "value": "$DB_ENDPOINT"},
        {"name": "REDIS_HOST", "value": "$REDIS_ENDPOINT"}
      ],
      "secrets": [
        {"name": "DB_PASSWORD", "valueFrom": "arn:aws:secretsmanager:af-south-1:322424815667:secret:master-ovasabi-db-password"},
        {"name": "REDIS_PASSWORD", "valueFrom": "arn:aws:secretsmanager:af-south-1:322424815667:secret:master-ovasabi-redis-password"}
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/master-ovasabi",
          "awslogs-region": "af-south-1",
          "awslogs-stream-prefix": "app"
        }
      }
    }
  ]
}
EOF

# Register the task definition
aws ecs register-task-definition \
  --cli-input-json file://task-definition-prod.json \
  --region af-south-1
```

### Step 6: Create ECS Service

```bash
# Get cluster name and networking info
CLUSTER_NAME=$(aws cloudformation describe-stacks \
  --stack-name master-ovasabi-infrastructure \
  --query 'Stacks[0].Outputs[?OutputKey==`ECSClusterName`].OutputValue' \
  --output text --region af-south-1)

# Get subnet IDs (public subnets for ALB access)
SUBNET_IDS=$(aws ec2 describe-subnets \
  --filters "Name=vpc-id,Values=$VPC_ID" "Name=tag:Name,Values=*public*" \
  --query 'Subnets[].SubnetId' \
  --output text --region af-south-1 | tr '\t' ',')

# Get security group ID
SECURITY_GROUP_ID=$(aws ec2 describe-security-groups \
  --filters "Name=vpc-id,Values=$VPC_ID" "Name=group-name,Values=*ECS*" \
  --query 'SecurityGroups[0].GroupId' \
  --output text --region af-south-1)

# Create ECS service
aws ecs create-service \
  --cluster "$CLUSTER_NAME" \
  --service-name master-ovasabi-service \
  --task-definition master-ovasabi-prod \
  --desired-count 2 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[$SUBNET_IDS],securityGroups=[$SECURITY_GROUP_ID],assignPublicIp=ENABLED}" \
  --load-balancers "targetGroupArn=arn:aws:elasticloadbalancing:af-south-1:322424815667:targetgroup/master-ovasabi-tg/...,containerName=app,containerPort=8080" \
  --region af-south-1
```

### Step 7: Run Database Migrations

```bash
# Create a one-time migration task
aws ecs run-task \
  --cluster "$CLUSTER_NAME" \
  --task-definition master-ovasabi-prod \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[$SUBNET_IDS],securityGroups=[$SECURITY_GROUP_ID],assignPublicIp=ENABLED}" \
  --overrides '{
    "containerOverrides": [{
      "name": "app",
      "command": ["/app/migrate", "up"]
    }]
  }' \
  --region af-south-1
```

## 🔧 Alternative: Using AWS Console

If you prefer using the AWS web console:

### 1. VPC Setup

- Go to VPC Console
- Create VPC with public/private subnets
- Set up Internet Gateway and NAT Gateway
- Configure route tables

### 2. RDS Setup

- Go to RDS Console
- Create PostgreSQL instance
- Use private subnets
- Note the endpoint

### 3. ElastiCache Setup

- Go to ElastiCache Console
- Create Redis cluster
- Use private subnets
- Note the endpoint

### 4. ECR Setup

- Go to ECR Console
- Create repositories for each service
- Push images using local Docker

### 5. ECS Setup

- Go to ECS Console
- Create cluster (Fargate)
- Create task definitions
- Create services
- Set up load balancer

## 🔍 Monitoring and Troubleshooting

### Check Service Status

```bash
# Check service status
aws ecs describe-services \
  --cluster "$CLUSTER_NAME" \
  --services master-ovasabi-service \
  --region af-south-1

# Check running tasks
aws ecs list-tasks \
  --cluster "$CLUSTER_NAME" \
  --service-name master-ovasabi-service \
  --region af-south-1

# View logs
aws logs tail /ecs/master-ovasabi --follow --region af-south-1
```

### Access Your Application

```bash
# Get load balancer DNS
ALB_DNS=$(aws cloudformation describe-stacks \
  --stack-name master-ovasabi-infrastructure \
  --query 'Stacks[0].Outputs[?OutputKey==`LoadBalancerDNS`].OutputValue' \
  --output text --region af-south-1)

echo "Your application is available at: http://$ALB_DNS"
```

## 🔒 Security Best Practices

1. **Use Secrets Manager** for all passwords
2. **Enable VPC Flow Logs** for network monitoring
3. **Set up CloudWatch Alarms** for monitoring
4. **Use IAM roles** instead of access keys
5. **Enable AWS CloudTrail** for audit logging
6. **Configure WAF** for web application firewall

## 💰 Cost Optimization

1. **Use Fargate Spot** for non-critical workloads
2. **Set up auto-scaling** based on CPU/memory
3. **Use Reserved Instances** for RDS
4. **Monitor costs** with AWS Cost Explorer
5. **Set up billing alerts**

## 🆘 Troubleshooting

### Common Issues

**Service won't start:**

- Check task definition for correct image URIs
- Verify security groups allow traffic
- Check logs in CloudWatch

**Can't connect to database:**

- Verify RDS security group allows ECS access
- Check database endpoint in environment variables
- Ensure database is in same VPC

**Images won't pull:**

- Verify ECR repository exists
- Check IAM permissions for ECS task execution role
- Ensure images are pushed to correct registry

## 📚 Additional Resources

- [AWS ECS Documentation](https://docs.aws.amazon.com/ecs/)
- [AWS Fargate Pricing](https://aws.amazon.com/fargate/pricing/)
- [ECS Best Practices](https://docs.aws.amazon.com/AmazonECS/latest/bestpracticesguide/introduction.html)
- [AWS Well-Architected Framework](https://aws.amazon.com/architecture/well-architected/)

---

Need help? Check the troubleshooting section or create an issue in the repository.
