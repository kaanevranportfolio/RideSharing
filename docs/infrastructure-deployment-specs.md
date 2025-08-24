# Infrastructure & Deployment Specifications

## 🏗️ **Production Infrastructure Implementation Guide**

This document provides comprehensive specifications for deploying the rideshare platform in a production-ready environment with full observability, security, and scalability.

---

## 🐳 **1. Kubernetes Deployment Architecture**

### **1.1 Namespace Organization**
```yaml
# infrastructure/kubernetes/namespaces/
apiVersion: v1
kind: Namespace
metadata:
  name: rideshare-prod
  labels:
    environment: production
    app: rideshare-platform
---
apiVersion: v1
kind: Namespace
metadata:
  name: rideshare-monitoring
  labels:
    environment: production
    purpose: monitoring
---
apiVersion: v1
kind: Namespace
metadata:
  name: rideshare-ingress
  labels:
    environment: production
    purpose: ingress
```

### **1.2 Service Deployment Strategy**
```yaml
# Example: User Service Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
  namespace: rideshare-prod
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: user-service
  template:
    metadata:
      labels:
        app: user-service
        version: v1.0.0
    spec:
      containers:
      - name: user-service
        image: rideshare/user-service:v1.0.0
        ports:
        - containerPort: 8051
          name: http
        - containerPort: 50051
          name: grpc
        env:
        - name: ENVIRONMENT
          value: "production"
        - name: DB_HOST
          valueFrom:
            secretKeyRef:
              name: postgres-credentials
              key: host
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8051
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8051
          initialDelaySeconds: 5
          periodSeconds: 5
```

### **1.3 Database StatefulSets**
```yaml
# PostgreSQL StatefulSet
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: rideshare-prod
spec:
  serviceName: postgres
  replicas: 3
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        env:
        - name: POSTGRES_DB
          value: rideshare
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: postgres-credentials
              key: username
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-credentials
              key: password
        - name: PGDATA
          value: /var/lib/postgresql/data/pgdata
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
  volumeClaimTemplates:
  - metadata:
      name: postgres-storage
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 20Gi
      storageClassName: fast-ssd
```

---

## 📊 **2. Monitoring Stack Implementation**

### **2.1 Prometheus Configuration**
```yaml
# infrastructure/monitoring/prometheus/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "rules/*.yml"

scrape_configs:
  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
    - role: pod
    relabel_configs:
    - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
      action: keep
      regex: true
    - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
      action: replace
      target_label: __metrics_path__
      regex: (.+)

  - job_name: 'rideshare-services'
    static_configs:
    - targets: 
      - 'user-service:8051'
      - 'vehicle-service:8052'
      - 'geo-service:8053'
      - 'matching-service:8054'
      - 'pricing-service:8055'
      - 'trip-service:8056'
      - 'payment-service:8057'
      - 'api-gateway:8080'

alerting:
  alertmanagers:
  - static_configs:
    - targets:
      - alertmanager:9093
```

### **2.2 Grafana Dashboards**
```json
{
  "dashboard": {
    "title": "Rideshare Platform - System Overview",
    "panels": [
      {
        "title": "Active Trips",
        "type": "stat",
        "targets": [
          {
            "expr": "sum(rideshare_active_trips_total)",
            "legendFormat": "Active Trips"
          }
        ]
      },
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{service}} - {{method}}"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          }
        ]
      },
      {
        "title": "Driver Matching Latency",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(rideshare_matching_duration_seconds_bucket[5m]))",
            "legendFormat": "Matching Time (95th percentile)"
          }
        ]
      }
    ]
  }
}
```

### **2.3 Jaeger Tracing Setup**
```yaml
# infrastructure/monitoring/jaeger/jaeger-deployment.yml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger
  namespace: rideshare-monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: jaeger
  template:
    metadata:
      labels:
        app: jaeger
    spec:
      containers:
      - name: jaeger
        image: jaegertracing/all-in-one:1.50
        env:
        - name: COLLECTOR_OTLP_ENABLED
          value: "true"
        - name: SPAN_STORAGE_TYPE
          value: "elasticsearch"
        - name: ES_SERVER_URLS
          value: "http://elasticsearch:9200"
        ports:
        - containerPort: 16686
          name: ui
        - containerPort: 14268
          name: collector
        - containerPort: 4317
          name: otlp-grpc
        - containerPort: 4318
          name: otlp-http
```

---

## 🔒 **3. Security Implementation**

### **3.1 Network Policies**
```yaml
# infrastructure/security/network-policies.yml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: rideshare-network-policy
  namespace: rideshare-prod
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: rideshare-ingress
    - podSelector:
        matchLabels:
          app: api-gateway
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: rideshare-prod
  - to: []
    ports:
    - protocol: TCP
      port: 5432  # PostgreSQL
    - protocol: TCP
      port: 27017 # MongoDB
    - protocol: TCP
      port: 6379  # Redis
```

### **3.2 RBAC Configuration**
```yaml
# infrastructure/security/rbac.yml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: rideshare-prod
  name: rideshare-service-role
rules:
- apiGroups: [""]
  resources: ["pods", "services", "endpoints"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: rideshare-service-binding
  namespace: rideshare-prod
subjects:
- kind: ServiceAccount
  name: rideshare-service-account
  namespace: rideshare-prod
roleRef:
  kind: Role
  name: rideshare-service-role
  apiGroup: rbac.authorization.k8s.io
```

### **3.3 Secret Management**
```yaml
# infrastructure/security/secrets.yml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-credentials
  namespace: rideshare-prod
type: Opaque
data:
  host: <base64-encoded-host>
  username: <base64-encoded-username>
  password: <base64-encoded-password>
---
apiVersion: v1
kind: Secret
metadata:
  name: jwt-secret
  namespace: rideshare-prod
type: Opaque
data:
  secret-key: <base64-encoded-jwt-secret>
---
apiVersion: v1
kind: Secret
metadata:
  name: redis-credentials
  namespace: rideshare-prod
type: Opaque
data:
  password: <base64-encoded-redis-password>
```

---

## 🚀 **4. CI/CD Pipeline Implementation**

### **4.1 GitHub Actions Workflow**
```yaml
# .github/workflows/ci-cd.yml
name: CI/CD Pipeline

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: rideshare-platform

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: test
          POSTGRES_DB: rideshare_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      redis:
        image: redis:7
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      mongodb:
        image: mongo:7
        env:
          MONGO_INITDB_ROOT_USERNAME: test
          MONGO_INITDB_ROOT_PASSWORD: test
        options: >-
          --health-cmd "mongosh --eval 'db.runCommand(\"ping\").ok'"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-
    
    - name: Download dependencies
      run: go mod download
    
    - name: Run unit tests
      run: go test -v -race -coverprofile=coverage.out ./...
    
    - name: Run integration tests
      run: |
        export DB_HOST=localhost
        export DB_PORT=5432
        export DB_NAME=rideshare_test
        export DB_USERNAME=postgres
        export DB_PASSWORD=test
        export REDIS_HOST=localhost
        export REDIS_PORT=6379
        export MONGO_HOST=localhost
        export MONGO_PORT=27017
        go test -v -tags=integration ./tests/integration/...
    
    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out

  build-and-push:
    needs: test
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    
    strategy:
      matrix:
        service: [user-service, vehicle-service, geo-service, matching-service, pricing-service, trip-service, payment-service, api-gateway]
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Log in to Container Registry
      uses: docker/login-action@v2
      with:
        registry: ${{ env.REGISTRY }}
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}
    
    - name: Extract metadata
      id: meta
      uses: docker/metadata-action@v4
      with:
        images: ${{ env.REGISTRY }}/${{ github.repository }}/${{ matrix.service }}
        tags: |
          type=ref,event=branch
          type=ref,event=pr
          type=sha,prefix={{branch}}-
          type=raw,value=latest,enable={{is_default_branch}}
    
    - name: Build and push Docker image
      uses: docker/build-push-action@v4
      with:
        context: ./services/${{ matrix.service }}
        push: true
        tags: ${{ steps.meta.outputs.tags }}
        labels: ${{ steps.meta.outputs.labels }}

  deploy:
    needs: build-and-push
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Kubectl
      uses: azure/setup-kubectl@v3
      with:
        version: 'v1.28.0'
    
    - name: Configure Kubernetes context
      run: |
        echo "${{ secrets.KUBE_CONFIG }}" | base64 -d > kubeconfig
        export KUBECONFIG=kubeconfig
    
    - name: Deploy to Kubernetes
      run: |
        export KUBECONFIG=kubeconfig
        kubectl apply -f infrastructure/kubernetes/
        kubectl rollout status deployment/user-service -n rideshare-prod
        kubectl rollout status deployment/vehicle-service -n rideshare-prod
        kubectl rollout status deployment/geo-service -n rideshare-prod
        kubectl rollout status deployment/matching-service -n rideshare-prod
        kubectl rollout status deployment/pricing-service -n rideshare-prod
        kubectl rollout status deployment/trip-service -n rideshare-prod
        kubectl rollout status deployment/payment-service -n rideshare-prod
        kubectl rollout status deployment/api-gateway -n rideshare-prod
```

### **4.2 Helm Chart Structure**
```yaml
# infrastructure/helm/rideshare-platform/Chart.yaml
apiVersion: v2
name: rideshare-platform
description: A Helm chart for Rideshare Platform
type: application
version: 1.0.0
appVersion: "1.0.0"

dependencies:
- name: postgresql
  version: "12.x.x"
  repository: "https://charts.bitnami.com/bitnami"
- name: mongodb
  version: "13.x.x"
  repository: "https://charts.bitnami.com/bitnami"
- name: redis
  version: "17.x.x"
  repository: "https://charts.bitnami.com/bitnami"
- name: prometheus
  version: "23.x.x"
  repository: "https://prometheus-community.github.io/helm-charts"
- name: grafana
  version: "6.x.x"
  repository: "https://grafana.github.io/helm-charts"
```

---

## 🔧 **5. Service Mesh Implementation (Istio)**

### **5.1 Istio Configuration**
```yaml
# infrastructure/service-mesh/istio-config.yml
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  name: rideshare-istio
spec:
  values:
    global:
      meshID: rideshare-mesh
      network: rideshare-network
  components:
    pilot:
      k8s:
        resources:
          requests:
            cpu: 200m
            memory: 128Mi
    ingressGateways:
    - name: istio-ingressgateway
      enabled: true
      k8s:
        service:
          type: LoadBalancer
```

### **5.2 Traffic Management**
```yaml
# infrastructure/service-mesh/virtual-service.yml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: rideshare-api
  namespace: rideshare-prod
spec:
  hosts:
  - api.rideshare.com
  gateways:
  - rideshare-gateway
  http:
  - match:
    - uri:
        prefix: /api/v1/users
    route:
    - destination:
        host: user-service
        port:
          number: 8051
  - match:
    - uri:
        prefix: /api/v1/vehicles
    route:
    - destination:
        host: vehicle-service
        port:
          number: 8052
  - match:
    - uri:
        prefix: /graphql
    route:
    - destination:
        host: api-gateway
        port:
          number: 8080
```

---

## 📈 **6. Auto-scaling Configuration**

### **6.1 Horizontal Pod Autoscaler**
```yaml
# infrastructure/scaling/hpa.yml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: user-service-hpa
  namespace: rideshare-prod
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: user-service
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  - type: Pods
    pods:
      metric:
        name: http_requests_per_second
      target:
        type: AverageValue
        averageValue: "100"
```

### **6.2 Vertical Pod Autoscaler**
```yaml
# infrastructure/scaling/vpa.yml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: user-service-vpa
  namespace: rideshare-prod
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: user-service
  updatePolicy:
    updateMode: "Auto"
  resourcePolicy:
    containerPolicies:
    - containerName: user-service
      maxAllowed:
        cpu: 2
        memory: 4Gi
      minAllowed:
        cpu: 100m
        memory: 128Mi
```

---

## 💾 **7. Data Backup & Disaster Recovery**

### **7.1 Database Backup Strategy**
```yaml
# infrastructure/backup/postgres-backup-cronjob.yml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
  namespace: rideshare-prod
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: postgres-backup
            image: postgres:15-alpine
            command:
            - /bin/bash
            - -c
            - |
              pg_dump -h postgres -U $POSTGRES_USER -d rideshare | gzip > /backup/rideshare-$(date +%Y%m%d-%H%M%S).sql.gz
              # Upload to S3 or other storage
              aws s3 cp /backup/rideshare-$(date +%Y%m%d-%H%M%S).sql.gz s3://rideshare-backups/postgres/
            env:
            - name: POSTGRES_USER
              valueFrom:
                secretKeyRef:
                  name: postgres-credentials
                  key: username
            - name: PGPASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-credentials
                  key: password
            volumeMounts:
            - name: backup-storage
              mountPath: /backup
          volumes:
          - name: backup-storage
            emptyDir: {}
          restartPolicy: OnFailure
```

### **7.2 Disaster Recovery Plan**
```bash
#!/bin/bash
# scripts/disaster-recovery.sh

# 1. Restore from backup
kubectl create namespace rideshare-recovery
kubectl apply -f infrastructure/kubernetes/secrets/ -n rideshare-recovery

# 2. Restore databases
kubectl run postgres-restore --image=postgres:15-alpine -n rideshare-recovery -- \
  bash -c "aws s3 cp s3://rideshare-backups/postgres/latest.sql.gz - | gunzip | psql -h postgres-recovery -U rideshare_user -d rideshare"

# 3. Deploy services in recovery namespace
helm install rideshare-recovery infrastructure/helm/rideshare-platform \
  --namespace rideshare-recovery \
  --set environment=recovery

# 4. Switch traffic to recovery environment
kubectl patch virtualservice rideshare-api -n rideshare-prod --type='json' \
  -p='[{"op": "replace", "path": "/spec/http/0/route/0/destination/host", "value": "api-gateway.rideshare-recovery.svc.cluster.local"}]'
```

This comprehensive infrastructure specification provides a production-ready deployment strategy with full observability, security, scalability, and disaster recovery capabilities for the rideshare platform.