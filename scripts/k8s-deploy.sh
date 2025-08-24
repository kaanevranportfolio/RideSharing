#!/bin/bash
# Kubernetes deployment script for Rideshare Platform

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

NAMESPACE="rideshare-platform"
KUBECTL_TIMEOUT="300s"

echo -e "${BLUE}🚀 Deploying Rideshare Platform to Kubernetes${NC}"
echo "=================================================="

# Function to check if kubectl is available
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        echo -e "${RED}❌ kubectl is not installed or not in PATH${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ kubectl is available${NC}"
}

# Function to check cluster connectivity
check_cluster() {
    if ! kubectl cluster-info &> /dev/null; then
        echo -e "${RED}❌ Cannot connect to Kubernetes cluster${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Connected to Kubernetes cluster${NC}"
}

# Function to apply manifests with retry
apply_manifest() {
    local file=$1
    local description=$2
    
    echo -e "${YELLOW}📦 Deploying $description...${NC}"
    
    if kubectl apply -f "$file" --timeout="$KUBECTL_TIMEOUT"; then
        echo -e "${GREEN}✅ $description deployed successfully${NC}"
    else
        echo -e "${RED}❌ Failed to deploy $description${NC}"
        return 1
    fi
}

# Function to wait for deployment
wait_for_deployment() {
    local deployment=$1
    local namespace=$2
    
    echo -e "${YELLOW}⏳ Waiting for $deployment to be ready...${NC}"
    
    if kubectl wait --for=condition=available --timeout="$KUBECTL_TIMEOUT" deployment/"$deployment" -n "$namespace"; then
        echo -e "${GREEN}✅ $deployment is ready${NC}"
    else
        echo -e "${RED}❌ $deployment failed to become ready${NC}"
        return 1
    fi
}

# Function to create namespace if it doesn't exist
create_namespace() {
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        echo -e "${YELLOW}📁 Namespace $NAMESPACE already exists${NC}"
    else
        echo -e "${YELLOW}📁 Creating namespace $NAMESPACE...${NC}"
        apply_manifest "k8s/namespace.yaml" "Namespace"
    fi
}

# Main deployment function
deploy() {
    echo -e "\n${BLUE}🔍 Pre-deployment checks${NC}"
    check_kubectl
    check_cluster
    
    echo -e "\n${BLUE}📁 Setting up namespace and configuration${NC}"
    create_namespace
    apply_manifest "k8s/configmap.yaml" "ConfigMaps and Secrets"
    
    echo -e "\n${BLUE}🗄️ Deploying databases${NC}"
    apply_manifest "k8s/database/postgres.yaml" "PostgreSQL Database"
    apply_manifest "k8s/database/mongodb.yaml" "MongoDB Database"
    apply_manifest "k8s/database/redis.yaml" "Redis Cache"
    
    # Wait for databases to be ready
    wait_for_deployment "postgres" "$NAMESPACE"
    wait_for_deployment "mongodb" "$NAMESPACE"
    wait_for_deployment "redis" "$NAMESPACE"
    
    echo -e "\n${BLUE}🚀 Deploying core services${NC}"
    apply_manifest "k8s/services/core-services.yaml" "Core Services (User, Vehicle)"
    apply_manifest "k8s/services/api-gateway.yaml" "API Gateway"
    
    # Wait for core services
    wait_for_deployment "user-service" "$NAMESPACE"
    wait_for_deployment "vehicle-service" "$NAMESPACE"
    wait_for_deployment "api-gateway" "$NAMESPACE"
    
    echo -e "\n${BLUE}📊 Deploying monitoring stack${NC}"
    apply_manifest "k8s/monitoring/prometheus.yaml" "Prometheus Monitoring"
    apply_manifest "k8s/monitoring/grafana.yaml" "Grafana Dashboards"
    
    # Wait for monitoring
    wait_for_deployment "prometheus" "$NAMESPACE"
    wait_for_deployment "grafana" "$NAMESPACE"
    
    echo -e "\n${BLUE}📈 Setting up auto-scaling${NC}"
    apply_manifest "k8s/autoscaling/hpa.yaml" "Horizontal Pod Autoscalers"
    
    echo -e "\n${GREEN}🎉 Deployment completed successfully!${NC}"
    
    # Display service information
    display_service_info
}

# Function to display service information
display_service_info() {
    echo -e "\n${BLUE}📋 Service Information${NC}"
    echo "======================"
    
    echo -e "\n${YELLOW}Services:${NC}"
    kubectl get services -n "$NAMESPACE" -o wide
    
    echo -e "\n${YELLOW}Pods:${NC}"
    kubectl get pods -n "$NAMESPACE" -o wide
    
    echo -e "\n${YELLOW}Ingress:${NC}"
    kubectl get ingress -n "$NAMESPACE"
    
    echo -e "\n${BLUE}🌐 Access URLs:${NC}"
    echo "API Gateway: http://api.rideshare.local (add to /etc/hosts)"
    echo "Grafana: kubectl port-forward svc/grafana 3000:3000 -n $NAMESPACE"
    echo "Prometheus: kubectl port-forward svc/prometheus 9090:9090 -n $NAMESPACE"
}

# Function to delete deployment
delete() {
    echo -e "${RED}🗑️ Deleting Rideshare Platform deployment${NC}"
    
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        kubectl delete namespace "$NAMESPACE" --timeout="$KUBECTL_TIMEOUT"
        echo -e "${GREEN}✅ Namespace $NAMESPACE deleted${NC}"
    else
        echo -e "${YELLOW}⚠️ Namespace $NAMESPACE does not exist${NC}"
    fi
}

# Function to show status
status() {
    echo -e "${BLUE}📊 Rideshare Platform Status${NC}"
    echo "============================="
    
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        echo -e "${GREEN}✅ Namespace exists${NC}"
        
        echo -e "\n${YELLOW}Deployments:${NC}"
        kubectl get deployments -n "$NAMESPACE"
        
        echo -e "\n${YELLOW}Pods:${NC}"
        kubectl get pods -n "$NAMESPACE"
        
        echo -e "\n${YELLOW}Services:${NC}"
        kubectl get services -n "$NAMESPACE"
    else
        echo -e "${RED}❌ Namespace $NAMESPACE does not exist${NC}"
    fi
}

# Function to show logs
logs() {
    local service=$1
    if [ -z "$service" ]; then
        echo -e "${RED}❌ Please specify a service name${NC}"
        echo "Available services: api-gateway, user-service, vehicle-service, prometheus, grafana"
        return 1
    fi
    
    echo -e "${BLUE}📋 Logs for $service${NC}"
    kubectl logs -f deployment/"$service" -n "$NAMESPACE"
}

# Function to scale service
scale() {
    local service=$1
    local replicas=$2
    
    if [ -z "$service" ] || [ -z "$replicas" ]; then
        echo -e "${RED}❌ Please specify service name and replica count${NC}"
        echo "Usage: $0 scale <service-name> <replica-count>"
        return 1
    fi
    
    echo -e "${YELLOW}📊 Scaling $service to $replicas replicas${NC}"
    kubectl scale deployment "$service" --replicas="$replicas" -n "$NAMESPACE"
}

# Main script logic
case "${1:-deploy}" in
    deploy)
        deploy
        ;;
    delete)
        delete
        ;;
    status)
        status
        ;;
    logs)
        logs "$2"
        ;;
    scale)
        scale "$2" "$3"
        ;;
    *)
        echo "Usage: $0 {deploy|delete|status|logs <service>|scale <service> <replicas>}"
        exit 1
        ;;
esac
