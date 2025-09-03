#!/bin/bash

# Posthoot Server Kubernetes Deployment Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="posthoot-server"
REGISTRY="your-registry"  # Change this to your registry
IMAGE_NAME="posthoot-server"
TAG="latest"

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed"
        exit 1
    fi
    
    # Check if kubectl is configured
    if ! kubectl cluster-info &> /dev/null; then
        log_error "kubectl is not configured or cluster is not accessible"
        exit 1
    fi
    
    # Check if docker is installed
    if ! command -v docker &> /dev/null; then
        log_error "docker is not installed"
        exit 1
    fi
    
    log_info "Prerequisites check passed"
}

build_and_push_image() {
    log_info "Building Docker image..."
    
    # Build the image
    docker build -t ${IMAGE_NAME}:${TAG} .
    
    # Tag for registry
    docker tag ${IMAGE_NAME}:${TAG} ${REGISTRY}/${IMAGE_NAME}:${TAG}
    
    log_info "Pushing image to registry..."
    docker push ${REGISTRY}/${IMAGE_NAME}:${TAG}
    
    log_info "Image built and pushed successfully"
}

update_image_in_deployment() {
    log_info "Updating image in deployment..."
    
    # Update the image in the deployment
    kubectl set image deployment/posthoot-server posthoot-server=${REGISTRY}/${IMAGE_NAME}:${TAG} -n ${NAMESPACE}
    
    log_info "Image updated in deployment"
}

deploy_resources() {
    log_info "Deploying Kubernetes resources..."
    
    # Create namespace if it doesn't exist
    kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -
    
    # Apply all resources using kustomize
    kubectl apply -k .
    
    log_info "Resources deployed successfully"
}

wait_for_deployment() {
    log_info "Waiting for deployment to be ready..."
    
    # Wait for all deployments to be ready
    kubectl wait --for=condition=available --timeout=300s deployment/posthoot-postgres -n ${NAMESPACE}
    kubectl wait --for=condition=available --timeout=300s deployment/posthoot-redis -n ${NAMESPACE}
    kubectl wait --for=condition=available --timeout=300s deployment/posthoot-server -n ${NAMESPACE}
    kubectl wait --for=condition=available --timeout=300s deployment/posthoot-asynqmon -n ${NAMESPACE}
    
    log_info "All deployments are ready"
}

show_status() {
    log_info "Deployment status:"
    echo ""
    kubectl get all -n ${NAMESPACE}
    echo ""
    kubectl get ingress -n ${NAMESPACE}
    echo ""
    kubectl get pvc -n ${NAMESPACE}
}

main() {
    log_info "Starting Posthoot Server deployment..."
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-build)
                SKIP_BUILD=true
                shift
                ;;
            --registry)
                REGISTRY="$2"
                shift 2
                ;;
            --tag)
                TAG="$2"
                shift 2
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  --skip-build    Skip building and pushing Docker image"
                echo "  --registry      Docker registry (default: your-registry)"
                echo "  --tag          Image tag (default: latest)"
                echo "  --help         Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    check_prerequisites
    
    if [[ "$SKIP_BUILD" != "true" ]]; then
        build_and_push_image
        update_image_in_deployment
    fi
    
    deploy_resources
    wait_for_deployment
    show_status
    
    log_info "Deployment completed successfully!"
    log_info "Your application should be available at: https://api.posthoot.com"
    log_info "Asynqmon dashboard: http://posthoot-asynqmon:8080"
}

# Run main function
main "$@"
