# AWS Deployment Quick Reference

## One-Command Deployment

To deploy your entire Master Ovasabi application to AWS, run this single command:

```bash
./deployments/aws/deploy-full.sh
```

This script will:

1. ✅ Check all prerequisites (AWS CLI, Docker, credentials)
2. 🔐 Generate secure passwords
3. 🏗️ Deploy AWS infrastructure (VPC, RDS, Redis, ECS, ALB)
4. 🐳 Build and push Docker images to ECR
5. 📋 Create ECS task definition
6. 🚀 Deploy the ECS service
7. ⏳ Wait for deployment to complete
8. 📊 Show deployment information

## Prerequisites

Before running the deployment script, ensure you have:

```bash
# 1. Configure AWS CLI (only needed once)
aws configure
# Enter your:
# - AWS Access Key ID
# - AWS Secret Access Key  
# - Default region: af-south-1
# - Default output format: json

# 2. Make sure Docker is running
docker --version

# 3. Ensure you're in the project root directory
cd /path/to/master-ovasabi
```

## Post-Deployment

After successful deployment, you'll get:

- **Application URL**: [http://your-alb-dns-name.af-south-1.elb.amazonaws.com](http://your-alb-dns-name.af-south-1.elb.amazonaws.com)
- **CloudWatch Logs**: View application logs in AWS Console
- **ECS Service**: Manage scaling and updates via AWS Console

## Quick Commands

```bash
# Check deployment status
aws ecs describe-services --cluster master-ovasabi-cluster --services master-ovasabi-service --region af-south-1

# View logs
aws logs tail /ecs/master-ovasabi --follow --region af-south-1

# Scale service (increase replicas)
aws ecs update-service --cluster master-ovasabi-cluster --service master-ovasabi-service --desired-count 2 --region af-south-1

# Delete everything (cleanup)
aws cloudformation delete-stack --stack-name master-ovasabi-infrastructure --region af-south-1
```

## Cost Estimate

Expected monthly costs for basic deployment:

- ECS Fargate (1 task): ~$15
- RDS PostgreSQL (db.t3.micro): ~$15  
- ElastiCache Redis (cache.t3.micro): ~$15
- Application Load Balancer: ~$20
- **Total: ~$65/month**

## Troubleshooting

### Common Issues

1. **AWS credentials not configured**

   ```bash
   aws configure
   ```

2. **Docker not running**

   ```bash
   docker info  # Should not error
   ```

3. **Region not supported**
   - Script uses `af-south-1` by default
   - Change `AWS_REGION` environment variable if needed

4. **Insufficient permissions**
   - Ensure your AWS user has permissions for ECS, ECR, RDS, VPC, CloudFormation

### Getting Help

```bash
# Check AWS CLI configuration
aws sts get-caller-identity

# Test Docker access
docker ps

# Validate environment variables
./scripts/validate-env.sh
```

## Manual Step-by-Step (Alternative)

If you prefer manual control, follow the detailed guide:

- See `deployments/aws/SETUP-GUIDE.md` for complete step-by-step instructions
- Use individual scripts in `deployments/aws/` directory
