#!/bin/bash

# OneWay Support Bot - Simple Deploy Script

set -e

echo "🚀 OneWay Support Bot - Deploy Script"
echo "======================================"

# Check if .env exists
if [ ! -f ".env" ]; then
    echo "⚠️  .env file not found!"
    echo "📋 Copy env.example to .env and configure it:"
    echo "   cp env.example .env"
    echo "   nano .env"
    exit 1
fi

# Check required environment variables
source .env
required_vars=("BOT_TOKEN" "CHANNEL_ID" "SUPERGROUP_ID" "MONGO_ROOT_PASSWORD")

for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        echo "❌ Missing required variable: $var"
        echo "📝 Please set it in .env file"
        exit 1
    fi
done

echo "✅ Environment variables OK"

# Stop existing containers
echo "🛑 Stopping existing containers..."
docker-compose down 2>/dev/null || true

# Pull latest images and build
echo "🔄 Building application..."
docker-compose build --no-cache

# Start services
echo "🚀 Starting services..."
docker-compose up -d

# Wait for services to be ready
echo "⏳ Waiting for services to start..."
sleep 10

# Check health
echo "🏥 Checking service health..."
if docker-compose ps | grep -q "unhealthy"; then
    echo "❌ Some services are unhealthy:"
    docker-compose ps
    echo ""
    echo "📋 Check logs with: docker-compose logs"
    exit 1
fi

echo ""
echo "🎉 Deployment completed successfully!"
echo ""
echo "📊 Service status:"
docker-compose ps
echo ""
echo "📋 Useful commands:"
echo "  docker-compose logs -f bot     # View bot logs"
echo "  docker-compose logs -f mongo   # View database logs" 
echo "  docker-compose ps              # Check status"
echo "  docker-compose down            # Stop all services"
echo ""
echo "🔗 Health check: curl http://localhost:8081/health"
echo "📈 Metrics: http://localhost:9090/metrics"
