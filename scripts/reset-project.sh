#!/bin/bash

# aiplusall-kb Project Reset Script
# This script will clean all debugging data and reset the project to initial state

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker first."
        exit 1
    fi
}

# Function to load environment variables
load_env() {
    if [ -f .env ]; then
        print_info "Loading environment variables from .env file..."
        export $(grep -v '^#' .env | xargs)
    else
        print_warning ".env file not found. Using default values."
    fi
}

# Function to stop all running containers
stop_containers() {
    print_info "Stopping all running containers..."
    
    # Stop using docker-compose if available
    if [ -f docker-compose.yml ]; then
        docker-compose down --remove-orphans || true
    fi
    
    if [ -f docker-compose.dev.yml ]; then
        docker-compose -f docker-compose.dev.yml down --remove-orphans || true
    fi
    
    # Stop individual containers that might be running
    containers=(
        "aiplusall-kb-app"
        "aiplusall-kb-docreader" 
        "aiplusall-kb-frontend"
        "postgres"
        "redis"
        "minio"
        "neo4j"
        "ollama"
    )
    
    for container in "${containers[@]}"; do
        if docker ps -q -f name="$container" | grep -q .; then
            print_info "Stopping container: $container"
            docker stop "$container" || true
        fi
    done
    
    print_success "All containers stopped"
}

# Function to remove Docker volumes
clean_docker_volumes() {
    print_info "Cleaning Docker volumes..."
    
    # Common volume patterns to clean
    volumes=(
        "aiplusall-kb_postgres-data"
        "aiplusall-kb_minio-data"
        "aiplusall-kb_redis-data"
        "aiplusall-kb_neo4j-data"
        "weknora_postgres-data"
        "weknora_minio_data"
        "weknora_redis_data"
        "weknora_neo4j_data"
    )
    
    for volume in "${volumes[@]}"; do
        if docker volume ls -q | grep -q "^${volume}$"; then
            print_info "Removing volume: $volume"
            docker volume rm "$volume" || true
        fi
    done
    
    # Remove any orphaned volumes
    print_info "Removing orphaned volumes..."
    docker volume prune -f || true
    
    print_success "Docker volumes cleaned"
}

# Function to clean local file storage
clean_local_storage() {
    print_info "Cleaning local file storage..."
    
    # Clean local storage directory from .env
    if [ -n "$LOCAL_STORAGE_BASE_DIR" ] && [ -d "$LOCAL_STORAGE_BASE_DIR" ]; then
        print_info "Cleaning local storage directory: $LOCAL_STORAGE_BASE_DIR"
        rm -rf "$LOCAL_STORAGE_BASE_DIR"/*
    fi
    
    # Clean common data directories
    data_dirs=(
        "./data"
        "./tmp"
        "./uploads"
        "./files"
        "./storage"
    )
    
    for dir in "${data_dirs[@]}"; do
        if [ -d "$dir" ]; then
            print_info "Cleaning directory: $dir"
            rm -rf "$dir"/*
        fi
    done
    
    print_success "Local storage cleaned"
}

# Function to clean temporary files
clean_temp_files() {
    print_info "Cleaning temporary files..."
    
    # Clean Go build cache
    if command -v go &> /dev/null; then
        go clean -cache -modcache -testcache || true
    fi
    
    # Clean common temp files
    find . -name "*.tmp" -type f -delete 2>/dev/null || true
    find . -name "*.log" -type f -delete 2>/dev/null || true
    find . -name ".DS_Store" -type f -delete 2>/dev/null || true
    
    # Clean frontend node_modules and build artifacts
    if [ -d "frontend/node_modules" ]; then
        print_info "Cleaning frontend node_modules..."
        rm -rf frontend/node_modules
    fi
    
    if [ -d "frontend/dist" ]; then
        print_info "Cleaning frontend build artifacts..."
        rm -rf frontend/dist
    fi
    
    # Clean Go binary
    if [ -f "aiplusall-kb" ]; then
        rm -f aiplusall-kb
    fi
    
    print_success "Temporary files cleaned"
}

# Function to reset database migrations
reset_migrations() {
    print_info "Database migrations will be reset on next startup..."
    # Note: We don't run migrations here as the database containers are stopped
    # Migrations will run fresh when containers are started again
    print_success "Migration state reset"
}

# Function to clean Docker images (optional)
clean_docker_images() {
    if [ "$1" = "--images" ]; then
        print_info "Cleaning Docker images..."
        
        # Remove project-specific images
        images=(
            "wechatopenai/aiplusall-kb-app"
            "wechatopenai/aiplusall-kb-docreader"
            "wechatopenai/aiplusall-kb-ui"
        )
        
        for image in "${images[@]}"; do
            if docker images -q "$image" | grep -q .; then
                print_info "Removing image: $image"
                docker rmi "$image" || true
            fi
        done
        
        # Clean dangling images
        docker image prune -f || true
        
        print_success "Docker images cleaned"
    fi
}

# Function to show reset summary
show_summary() {
    print_info "Reset Summary:"
    echo "  ✓ All containers stopped"
    echo "  ✓ Docker volumes removed"
    echo "  ✓ Local storage cleaned"
    echo "  ✓ Temporary files removed"
    echo "  ✓ Migration state reset"
    
    if [ "$1" = "--images" ]; then
        echo "  ✓ Docker images cleaned"
    fi
    
    echo ""
    print_success "Project has been reset to initial state!"
    echo ""
    print_info "To start fresh:"
    echo "  1. Run: make dev-start    (to start infrastructure)"
    echo "  2. Run: make migrate-up   (to setup database)"
    echo "  3. Run: make dev-app      (to start backend)"
    echo "  4. Run: make dev-frontend (to start frontend)"
    echo ""
    print_info "Or use one-click start: ./scripts/quick-dev.sh"
}

# Main execution
main() {
    print_info "Starting aiplusall-kb project reset..."
    echo ""
    
    # Confirmation prompt
    read -p "This will delete ALL data and reset the project to initial state. Continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Reset cancelled."
        exit 0
    fi
    
    echo ""
    
    # Check prerequisites
    check_docker
    load_env
    
    # Perform cleanup
    stop_containers
    clean_docker_volumes
    clean_local_storage
    clean_temp_files
    reset_migrations
    clean_docker_images "$1"
    
    echo ""
    show_summary "$1"
}

# Handle command line arguments
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "aiplusall-kb Project Reset Script"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --images    Also remove Docker images"
    echo "  --help      Show this help message"
    echo ""
    echo "This script will:"
    echo "  - Stop all running containers"
    echo "  - Remove all Docker volumes (database, storage, cache)"
    echo "  - Clean local file storage"
    echo "  - Remove temporary files"
    echo "  - Reset migration state"
    echo "  - Optionally remove Docker images"
    exit 0
fi

# Run main function
main "$1"