#!/bin/bash

# Simple EC2 Deployment Script for Master Ovasabi
# This script helps deploy your application to a single EC2 instance

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

# Generate EC2 user data script
generate_user_data() {
    print_step "Generating EC2 user data script..."
    
    cat > deployments/aws/ec2-user-data.sh << 'EOF'
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
EOF
    
    chmod +x deployments/aws/ec2-user-data.sh
    print_status "EC2 user data script created at: deployments/aws/ec2-user-data.sh"
}

# Generate CloudFormation template for simple EC2 deployment
generate_ec2_cloudformation() {
    print_step "Generating CloudFormation template for EC2 deployment..."
    
    cat > deployments/aws/ec2-simple.yaml << 'EOF'
AWSTemplateFormatVersion: '2010-09-09'
Description: 'Simple EC2 deployment for Master Ovasabi'

Parameters:
  InstanceType:
    Type: String
    Default: t3.medium
    AllowedValues: [t3.small, t3.medium, t3.large, t3.xlarge]
    Description: EC2 instance type
  
  KeyPairName:
    Type: AWS::EC2::KeyPair::KeyName
    Description: EC2 Key Pair for SSH access
  
  SSHLocation:
    Type: String
    Default: '0.0.0.0/0'
    Description: IP address range for SSH access (default allows all)

Resources:
  VPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: '10.0.0.0/16'
      EnableDnsHostnames: true
      EnableDnsSupport: true
      Tags:
        - Key: Name
          Value: !Sub '${AWS::StackName}-vpc'

  PublicSubnet:
    Type: AWS::EC2::Subnet
    Properties:
      VpcId: !Ref VPC
      CidrBlock: '10.0.1.0/24'
      AvailabilityZone: !Select [0, !GetAZs '']
      MapPublicIpOnLaunch: true
      Tags:
        - Key: Name
          Value: !Sub '${AWS::StackName}-public-subnet'

  InternetGateway:
    Type: AWS::EC2::InternetGateway
    Properties:
      Tags:
        - Key: Name
          Value: !Sub '${AWS::StackName}-igw'

  AttachGateway:
    Type: AWS::EC2::VPCGatewayAttachment
    Properties:
      VpcId: !Ref VPC
      InternetGatewayId: !Ref InternetGateway

  PublicRouteTable:
    Type: AWS::EC2::RouteTable
    Properties:
      VpcId: !Ref VPC
      Tags:
        - Key: Name
          Value: !Sub '${AWS::StackName}-public-rt'

  PublicRoute:
    Type: AWS::EC2::Route
    DependsOn: AttachGateway
    Properties:
      RouteTableId: !Ref PublicRouteTable
      DestinationCidrBlock: '0.0.0.0/0'
      GatewayId: !Ref InternetGateway

  PublicSubnetRouteTableAssociation:
    Type: AWS::EC2::SubnetRouteTableAssociation
    Properties:
      SubnetId: !Ref PublicSubnet
      RouteTableId: !Ref PublicRouteTable

  SecurityGroup:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupDescription: Security group for Master Ovasabi EC2 instance
      VpcId: !Ref VPC
      SecurityGroupIngress:
        - IpProtocol: tcp
          FromPort: 22
          ToPort: 22
          CidrIp: !Ref SSHLocation
        - IpProtocol: tcp
          FromPort: 80
          ToPort: 80
          CidrIp: '0.0.0.0/0'
        - IpProtocol: tcp
          FromPort: 443
          ToPort: 443
          CidrIp: '0.0.0.0/0'
        - IpProtocol: tcp
          FromPort: 8080
          ToPort: 8090
          CidrIp: '0.0.0.0/0'
        - IpProtocol: tcp
          FromPort: 50051
          ToPort: 50052
          CidrIp: '0.0.0.0/0'

  EC2Instance:
    Type: AWS::EC2::Instance
    Properties:
      InstanceType: !Ref InstanceType
      KeyName: !Ref KeyPairName
      ImageId: ami-0c94855ba95b798c7  # Amazon Linux 2023 (update for your region)
      SubnetId: !Ref PublicSubnet
      SecurityGroupIds:
        - !Ref SecurityGroup
      UserData:
        Fn::Base64: !Sub |
          #!/bin/bash
          yum update -y
          yum install -y docker git
          systemctl start docker
          systemctl enable docker
          usermod -a -G docker ec2-user
          curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
          chmod +x /usr/local/bin/docker-compose
          ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose
          mkdir -p /opt/master-ovasabi
          chown ec2-user:ec2-user /opt/master-ovasabi
      Tags:
        - Key: Name
          Value: !Sub '${AWS::StackName}-instance'

Outputs:
  InstancePublicIP:
    Description: Public IP address of the EC2 instance
    Value: !GetAtt EC2Instance.PublicIp
  
  InstancePublicDNS:
    Description: Public DNS name of the EC2 instance
    Value: !GetAtt EC2Instance.PublicDnsName
  
  SSHCommand:
    Description: Command to SSH into the instance
    Value: !Sub 'ssh -i ${KeyPairName}.pem ec2-user@${EC2Instance.PublicIp}'
EOF
    
    print_status "CloudFormation template created at: deployments/aws/ec2-simple.yaml"
}

# Generate deployment instructions
generate_instructions() {
    print_step "Generating deployment instructions..."
    
    cat > deployments/aws/README-EC2-DEPLOYMENT.md << 'EOF'
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
EOF
    
    print_status "Deployment instructions created at: deployments/aws/README-EC2-DEPLOYMENT.md"
}

# Main function
main() {
    echo "=== EC2 Deployment Setup ==="
    echo "Generating files for simple EC2 deployment"
    echo
    
    # Create aws directory if it doesn't exist
    mkdir -p deployments/aws
    
    generate_user_data
    generate_ec2_cloudformation
    generate_instructions
    
    echo
    print_step "Setup complete!"
    echo
    echo "=== Generated Files ==="
    echo "📄 deployments/aws/ec2-user-data.sh - EC2 setup script"
    echo "📄 deployments/aws/ec2-simple.yaml - CloudFormation template"
    echo "📄 deployments/aws/README-EC2-DEPLOYMENT.md - Deployment guide"
    echo
    echo "=== Quick Start ==="
    echo "1. Read the deployment guide: deployments/aws/README-EC2-DEPLOYMENT.md"
    echo "2. Create an EC2 key pair in AWS Console"
    echo "3. Deploy with CloudFormation"
    echo "4. SSH to your instance and set up the application"
    echo
    echo "This is the simplest way to get started on AWS!"
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
