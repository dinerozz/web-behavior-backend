#!/bin/bash
# deploy.sh

set -e

echo "🚀 Deploying expense-tracker to Kubernetes..."

if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl not found. Install kubectl."
    exit 1
fi

echo "📋 Checking cluster connection..."
kubectl cluster-info

echo "📁 Creating namespace..."
kubectl apply -f k3s/namespace.yaml

echo "🔐 apply secrets and configmap..."
kubectl apply -f k3s/secret.yaml
kubectl apply -f k3s/configmap.yaml

echo "🐘 Deploying PostgreSQL..."
kubectl apply -f k3s/postgres-pvc.yaml
kubectl apply -f k3s/postgres-deployment.yaml
kubectl apply -f k3s/postgres-service.yaml

echo "⏳ Waiting for PostgreSQL..."
kubectl rollout status deployment/postgres -n expense-tracker

echo "🏗️ Deploying backend..."
kubectl apply -f k3s/backend-deployment.yaml
kubectl apply -f k3s/backend-service.yaml

echo "⏳ Waiting for backend..."
kubectl rollout status deployment/expense-tracker-backend -n expense-tracker

echo "🌐 Configuring ingress..."
kubectl apply -f k3s/ingress.yaml


echo "📊 Deploy status:"
kubectl get pods -n expense-tracker
kubectl get services -n expense-tracker
kubectl get ingress -n expense-tracker

echo "✅ Deploy succeeded!"
echo "🔗 available on: https://web-behavior.space"
echo "💡 Add in /etc/hosts: echo '127.0.0.1 expense-tracker.local' | sudo tee -a /etc/hosts"