# StudentSystemGo Helm Chart

Deploys the full StudentSystemGo stack (Go API + MySQL primary/replica + Redis + RabbitMQ + worker) on Kubernetes.

## Quick start

```bash
# 1. Copy and fill in secrets
cp secrets.values.example.yaml secrets.values.yaml
$EDITOR secrets.values.yaml  # set real passwords

# 2. Install
helm install ssg . \
  --namespace studentsystemgo --create-namespace \
  -f values.yaml \
  -f secrets.values.yaml

# 3. Verify
kubectl get pods -n studentsystemgo