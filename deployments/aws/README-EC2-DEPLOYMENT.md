# EC2 Deployment Guide for Master Ovasabi

This guide helps you deploy Master Ovasabi on a single EC2 instance - the simplest AWS deployment option.

## Prerequisites

1. AWS CLI installed and configured
2. An EC2 Key Pair created in your AWS account
3. Basic knowledge of AWS EC2

## Step 1: Deploy Infrastructure

Deploy the CloudFormation stack:

```bash
aws cloudformation deploy \
  --template-file deployments/aws/ec2-simple.yaml \
  --stack-name master-ovasabi-ec2 \
  --parameter-overrides \
    KeyPairName=your-key-pair-name \
    InstanceType=t3.medium \
  --region us-east-1
```

## Step 2: Get Instance Information

```bash
# Get the public IP
aws cloudformation describe-stacks \
  --stack-name master-ovasabi-ec2 \
  --query 'Stacks[0].Outputs[?OutputKey==`InstancePublicIP`].OutputValue' \
  --output text
```

## Step 3: Connect to Your Instance

```bash
ssh -i your-key-pair.pem ec2-user@YOUR_INSTANCE_IP
```

## Step 4: Set Up Your Application

On the EC2 instance:

```bash
# Clone your repository
cd /opt/master-ovasabi
git clone https://github.com/nmxmxh/master-ovasabi.git .

# Create and configure .env file
cp .env.example .env
nano .env  # Edit with your actual values

# Build and start the application
docker-compose up -d
```

## Step 5: Access Your Application

Your application will be available at:
- Main app: http://YOUR_INSTANCE_IP:8080
- Nexus: http://YOUR_INSTANCE_IP:50052
- Media streaming: http://YOUR_INSTANCE_IP:8085
- WebSocket gateway: http://YOUR_INSTANCE_IP:8090

## Security Notes

- The default security group allows access from anywhere (0.0.0.0/0)
- For production, restrict SSH access to your IP only
- Consider setting up SSL certificates for HTTPS
- Use AWS Secrets Manager for sensitive configuration

## Monitoring and Logs

```bash
# View application logs
docker-compose logs -f

# Check container status
docker-compose ps

# Restart services
docker-compose restart
```

## Cleanup

To delete everything:

```bash
aws cloudformation delete-stack --stack-name master-ovasabi-ec2
```

## Next Steps

- Set up a custom domain with Route 53
- Configure SSL with Let's Encrypt or AWS Certificate Manager
- Set up CloudWatch monitoring
- Consider moving to ECS for better scalability
