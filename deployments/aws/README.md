# AWS Deployment Guide for Master Ovasabi

This guide provides multiple options for deploying your Master Ovasabi application on AWS, from simple to production-ready.

## 🏁 Quick Start (Recommended for Testing)

### Option 1: Single EC2 Instance

- ✅ Simplest to set up
- ✅ Good for development/testing
- ❌ Not highly available
- ❌ Manual scaling

```bash
# Generate EC2 deployment files
./deployments/aws/setup-ec2.sh

# Follow the instructions in:
# deployments/aws/README-EC2-DEPLOYMENT.md
```

## 🚀 Production Deployment Options

### Option 2: AWS ECS with Fargate

- ✅ Serverless containers
- ✅ Auto-scaling
- ✅ High availability
- ✅ Managed infrastructure

```bash
# Step 1: Build and push images to ECR
./deployments/aws/deploy-ecr.sh

# Step 2: Deploy infrastructure (manual CloudFormation/Terraform)
# Step 3: Create ECS services
```

### Option 3: AWS EKS (Kubernetes)

- ✅ Full Kubernetes features
- ✅ Multi-cloud portability
- ❌ More complex setup
- ❌ Higher learning curve

## 📋 Prerequisites

Before deploying to AWS, ensure you have:

1. **AWS Account** with appropriate permissions
2. **AWS CLI** installed and configured
3. **Docker** installed locally
4. **Environment variables** set up

### Set Up Environment Variables

1. Copy the example environment file:

   ```bash
   cp .env.example .env
   ```

2. Edit `.env` with your values:

   ```bash
   # Required for AWS deployment
   AWS_REGION=us-east-1
   AWS_ACCOUNT_ID=123456789012
   
   # Application configuration
   POSTGRES_PASSWORD=your_secure_password
   REDIS_PASSWORD=your_secure_password
   ADMIN_PASSWORD=your_secure_password
   ```

## 🔧 Fix the Docker Build Issue

Your original build error was due to the distroless base image. This has been fixed in the Dockerfile, but you can verify:

```bash
# Check the current Dockerfile.nexus
grep -n "FROM.*distroless" deployments/docker/Dockerfile.nexus

# Should show: FROM gcr.io/distroless/base-debian12:latest
```

## 🐳 Container Registry Options

### AWS ECR (Recommended)

```bash
# Use the provided script
./deployments/aws/deploy-ecr.sh
```

### Docker Hub (Alternative)

```bash
# Tag images for Docker Hub
docker compose -f deployments/docker/docker-compose.yml build
docker tag master-ovasabi-app:latest yourusername/master-ovasabi:latest
docker push yourusername/master-ovasabi:latest
```

## 🏗️ Infrastructure Components

Your application requires:

- **Compute**: ECS/EC2 for running containers
- **Database**: RDS PostgreSQL
- **Cache**: ElastiCache Redis
- **Load Balancer**: Application Load Balancer
- **Storage**: EFS/EBS for persistent data
- **Networking**: VPC, subnets, security groups
- **Secrets**: AWS Secrets Manager
- **Monitoring**: CloudWatch

## 🔐 Security Best Practices

1. **Use AWS Secrets Manager** for passwords
2. **Restrict security groups** to necessary ports only
3. **Use IAM roles** instead of access keys
4. **Enable encryption** at rest and in transit
5. **Set up VPC** with private subnets for databases
6. **Use HTTPS** with SSL certificates

## 📊 Cost Optimization

### Development/Testing

- Use `t3.micro` or `t3.small` instances
- Single AZ deployment
- Smaller RDS instances (db.t3.micro)

### Production

- Use multiple AZs for high availability
- Auto Scaling Groups for cost efficiency
- Reserved Instances for predictable workloads
- CloudWatch for monitoring and optimization

## 🚦 Deployment Steps Summary

### For EC2 (Simple)

1. Run `./deployments/aws/setup-ec2.sh`
2. Deploy CloudFormation stack
3. SSH to instance and set up application
4. Access via public IP

### For ECS (Production)

1. Run `./deployments/aws/deploy-ecr.sh` to push images
2. Deploy infrastructure (VPC, RDS, ElastiCache)
3. Create ECS cluster and services
4. Set up load balancer
5. Configure domain and SSL

## 🆘 Troubleshooting

### Build Issues

```bash
# If Docker build fails, check base images
docker pull gcr.io/distroless/base-debian12:latest

# If network issues, try different base image
# Edit Dockerfile.nexus: FROM alpine:latest
```

### Environment Variables

```bash
# Check current environment
env | grep -E "(AWS|DB|REDIS|ADMIN)"

# Load from .env file
set -a; source .env; set +a
```

### AWS Connectivity

```bash
# Test AWS CLI
aws sts get-caller-identity

# Test ECR access
aws ecr get-login-password --region us-east-1
```

## 📚 Additional Resources

- [AWS ECS Documentation](https://docs.aws.amazon.com/ecs/)
- [AWS ECR Documentation](https://docs.aws.amazon.com/ecr/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [AWS CloudFormation Documentation](https://docs.aws.amazon.com/cloudformation/)

## 🎯 Next Steps

1. **Choose your deployment option** based on your needs
2. **Set up environment variables** in `.env`
3. **Test locally** with `docker compose up`
4. **Deploy to AWS** using one of the provided methods
5. **Monitor and optimize** your deployment

For questions or issues, check the troubleshooting section or create an issue in the repository.
