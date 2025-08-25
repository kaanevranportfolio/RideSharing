# Security Configuration Guide

## 🔐 Secrets Management

This document outlines the secure configuration of sensitive values in the rideshare platform.

### ⚠️ IMPORTANT SECURITY NOTICE

All hardcoded passwords and secrets have been removed from this repository. You MUST configure proper secrets before deployment.

## Required Environment Variables

### 1. Database Credentials

```bash
# PostgreSQL (Required)
POSTGRES_PASSWORD=your-strong-password-here

# MongoDB (Required)
MONGODB_PASSWORD=your-strong-mongodb-password

# Redis (Optional)
REDIS_PASSWORD=your-redis-password-if-auth-enabled
```

### 2. JWT Secret

Generate a strong JWT secret:
```bash
# Generate a secure random key
openssl rand -base64 32
```

Set the JWT secret:
```bash
JWT_SECRET_KEY=your-generated-jwt-secret-here
```

### 3. Test Environment

For testing, set different credentials:
```bash
TEST_POSTGRES_PASSWORD=your-test-postgres-password
TEST_MONGODB_PASSWORD=your-test-mongodb-password
TEST_JWT_SECRET=your-test-jwt-secret
```

## Configuration Steps

### For Docker Compose

1. Copy the environment template:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` and replace all `CHANGE_ME_*` values with secure passwords

3. Generate and set strong passwords:
   ```bash
   # Example for generating passwords
   openssl rand -base64 20  # For database passwords
   openssl rand -base64 32  # For JWT secret
   ```

### For Kubernetes

1. Create base64 encoded secrets:
   ```bash
   echo -n "your-postgres-password" | base64
   echo -n "your-jwt-secret" | base64
   ```

2. Update `deployments/k8s/configmap.yaml`:
   - Replace `CHANGE_ME_BASE64_ENCODED` with your encoded PostgreSQL password
   - Replace `CHANGE_ME_BASE64_ENCODED_JWT_SECRET` with your encoded JWT secret

### For Helm

1. Create a values override file:
   ```bash
   cp deployments/helm/rideshare-platform/values.yaml values-production.yaml
   ```

2. Update the secrets in `values-production.yaml`:
   ```yaml
   postgresql:
     auth:
       postgresPassword: "your-strong-postgres-password"
   
   secrets:
     jwtSecret: "your-strong-jwt-secret"
   ```

3. Deploy with the override:
   ```bash
   helm install rideshare-platform ./deployments/helm/rideshare-platform -f values-production.yaml
   ```

## Security Best Practices

### Password Requirements
- Minimum 16 characters
- Mix of uppercase, lowercase, numbers, and symbols
- Unique for each service
- Never commit to version control

### JWT Secrets
- Minimum 32 characters
- Use cryptographically secure random generation
- Rotate regularly (every 90 days recommended)

### Environment-Specific Secrets
- Use different secrets for development, testing, and production
- Never use production secrets in development/testing

## GitGuardian Compliance

All changes made to address GitGuardian alerts:

✅ Removed hardcoded passwords from all configuration files
✅ Replaced hardcoded JWT secrets with environment variables
✅ Added proper environment variable templates
✅ Created secure configuration documentation
✅ Updated Docker Compose to use environment variables
✅ Updated Kubernetes manifests to use proper secrets
✅ Updated Helm charts to use configurable values

## Quick Setup Commands

For development:
```bash
# Set required environment variables
export POSTGRES_PASSWORD=$(openssl rand -base64 20)
export MONGODB_PASSWORD=$(openssl rand -base64 20)
export JWT_SECRET_KEY=$(openssl rand -base64 32)

# Save to .env file
cat > .env << EOF
POSTGRES_PASSWORD=$POSTGRES_PASSWORD
MONGODB_PASSWORD=$MONGODB_PASSWORD
JWT_SECRET_KEY=$JWT_SECRET_KEY
POSTGRES_USER=rideshare_user
POSTGRES_DB=rideshare
MONGODB_USER=rideshare_user
MONGODB_DB=rideshare_geo
EOF
```

For production deployment, use a proper secrets management solution like:
- HashiCorp Vault
- AWS Secrets Manager
- Azure Key Vault
- Google Secret Manager
- Kubernetes Secrets with external secret operators

## Verification

After configuration, verify no hardcoded secrets remain:
```bash
# Search for potential hardcoded values
grep -r "password\|secret\|key" --exclude-dir=node_modules --exclude-dir=.git --exclude="*.md" .
```

The search should only return references to environment variables, not actual secret values.
