# Harbor

Crater relies on an internal [Harbor](https://goharbor.io/) registry to support image building, storage, and secure distribution within the cluster. Harbor acts as the centralized container image repository for both development and deployment needs.

---

## Deployment

We recommend using the [official Harbor Helm Chart](https://github.com/goharbor/harbor-helm) for deployment. A basic example:

```bash
helm repo add harbor https://helm.goharbor.io
helm pull harbor/harbor --version 1.14.0
# Modify values.yaml as needed (see below)
helm install crater-harbor harbor/harbor -f values.yaml -n harbor-system --create-namespace
```

## Note: Ensure that:

The domain name (e.g., crater-harbor.act.buaa.edu.cn) is resolvable within your cluster or internal DNS.

You have a valid TLS setup or set expose.tls.enabled=false for HTTP.

## Configuration Tips
Set externalURL correctly in values.yaml:
```yaml
externalURL: https://crater-harbor.act.buaa.edu.cn
```
Configure ingress with appropriate class (ingress-nginx) and hostname.

Use internal certificates or disable TLS for testing:
```yaml
expose:
  tls:
    enabled: false
```

## Image Pull Secrets
To allow Crater components to pull images from Harbor:
```bash
kubectl create secret docker-registry crater-harbor-auth \
  --docker-server=crater-harbor.act.buaa.edu.cn \
  --docker-username=<user> \
  --docker-password=<password> \
  -n <target-namespace>
```
You can then reference this secret in your pod.spec.imagePullSecrets.

## Harbor Projects Structure
Projects are organized as follows:
| Project       | Purpose                    |
| ------------- | -------------------------- |
| crater/base   | Base OS and utilities      |
| crater/jobs   | User-submitted job images  |
| crater/charts | Mirrors for Helm charts    |
| crater/models | Large models for inference |

## Integration with Crater
Most system components (e.g., gpu-operator, ingress-nginx, prometheus) are preconfigured to use Harbor mirrors.

The image path should look like:
```ruby
crater-harbor.act.buaa.edu.cn/crater/<component>:<tag>
```
During Helm chart customization, update image.repository and imagePullSecrets.

## References
* [Harbor Documentation](https://goharbor.io/docs/)