# HAProxy Cloud Controller Manager Helm Chart

This Helm chart deploys the HAProxy Cloud Controller Manager (CCM) for Kubernetes.

## Configuration

### Basic Configuration

The chart supports configuration through the `values.yaml` file. Here are the key configuration options:

### Image Configuration

```yaml
image:
  name: bearsan/haproxy-ccm
  tag: 0.1.0
  useImagePullSecret:
    enabled: false
    name: ""
```

### Environment Variables

```yaml
env:
  useExistsSecret:
    enabled: false
    secretRef:
      baseUrl:
        name: ""
        key: ""
      auth:
        name: ""
        key: ""
  baseUrl: ""
  auth: ""
```

### Command Line Arguments

You can customize the command line arguments passed to the HAProxy CCM:

```yaml
args:
  # Cloud provider specific arguments
  cloudProvider: "haproxy"
  
  # Additional custom arguments (list format)
  additional: []
```

## Usage Examples

### Basic Deployment

```bash
helm install haproxy-ccm ./deploy/haproxy-ccm \
  --set env.baseUrl="http://haproxy:5555" \
  --set env.auth="admin:password"
```

### Deployment with Additional Arguments

```yaml
# custom-values.yaml
env:
  baseUrl: "grpc://haproxy-manager:50051"

args:
  additional:
    - "--log-level=debug"
    - "--bind-address=0.0.0.0"
    - "--port=10258"
    - "--haproxy-endpoint=grpc://haproxy-manager:50051"
```

```bash
helm install haproxy-ccm ./deploy/haproxy-ccm -f custom-values.yaml
```

### Using Existing Secrets

```yaml
# secret-values.yaml
env:
  useExistsSecret:
    enabled: true
    secretRef:
      baseUrl:
        name: "haproxy-config"
        key: "endpoint"
      auth:
        name: "haproxy-config"
        key: "auth"

args:
  additional:
    - "--v=4"
    - "--leader-elect=true"
```

```bash
helm install haproxy-ccm ./deploy/haproxy-ccm -f secret-values.yaml
```

## Common Additional Arguments

Here are some commonly used additional arguments:

- `--log-level=debug`: Set log level to debug
- `--bind-address=0.0.0.0`: Bind address for the CCM server
- `--port=10258`: Port for the CCM server
- `--haproxy-endpoint=<endpoint>`: Override HAProxy endpoint (alternative to env var)
- `--v=4`: Set verbosity level
- `--leader-elect=true`: Enable leader election for HA deployments
- `--cloud-config=<path>`: Path to cloud configuration file
- `--allocate-node-cidrs=false`: Disable node CIDR allocation
- `--configure-cloud-routes=false`: Disable cloud route configuration

## Installation

1. Clone the repository
2. Configure your `values.yaml` or use `--set` flags
3. Install the chart:

```bash
helm install haproxy-ccm ./deploy/haproxy-ccm
```

## Upgrading

```bash
helm upgrade haproxy-ccm ./deploy/haproxy-ccm
```

## Uninstallation

```bash
helm uninstall haproxy-ccm
```