# Azure Kubernetes Service deployment

This procedure creates billable Azure resources. Review the cost notes and do
not run the resource-creation commands until you are ready to incur charges.
The application images are built and pushed from Windows PowerShell using
Docker Desktop; the Dev Container does not need Docker, `kubectl`, or `kind`.

## What is deployed

- Azure Kubernetes Service with two `Standard_D2s_v5` worker nodes.
- Azure Container Registry (Basic) with the Raft and parking-web images.
- The Helm chart: three Raft pods, a parking-web pod, and the Raft headless
  service.
- An NGINX ingress controller with an Azure public load balancer.
- cert-manager and a Let's Encrypt ClusterIssuer for HTTPS.
- Optional Prometheus and Grafana deployments.

The browser/API app and Raft cluster are separate. Parking state is in memory
inside the parking-web pod and is not replicated through Raft or persisted.
Do not run multiple parking-web replicas when testing operations: separate
replicas would have separate state. This is an educational demo, not a
production parking system.

## Prerequisites on Windows

Install and sign in to:

- Azure CLI (`az`)
- Docker Desktop, running Linux containers
- Helm 3
- `kubectl` (or run `az aks install-cli` below)

In PowerShell, verify tools and authenticate. Select the subscription that
should own the resources:

```powershell
az login
az account list --output table
az account set --subscription "<YOUR_SUBSCRIPTION_ID>"
docker version
helm version
```

## Create Azure resources

Choose a supported Azure region. `eastus` is the example region below; VM
availability and prices vary. The random ACR and DNS names reduce name
collisions. Save the variable values for the remaining commands in this
PowerShell session.

```powershell
$Location = "eastus"
$ResourceGroup = "rg-smart-parking-demo"
$AcrName = "smartparking$((Get-Random -Minimum 10000 -Maximum 99999))"
$ClusterName = "aks-smart-parking-demo"
$DnsLabel = "smartparking$((Get-Random -Minimum 10000 -Maximum 99999))"

az group create `
  --name $ResourceGroup `
  --location $Location

az acr create `
  --resource-group $ResourceGroup `
  --name $AcrName `
  --sku Basic `
  --admin-enabled false

az aks create `
  --resource-group $ResourceGroup `
  --name $ClusterName `
  --location $Location `
  --tier free `
  --node-count 2 `
  --node-vm-size Standard_D2s_v5 `
  --attach-acr $AcrName `
  --generate-ssh-keys

az aks get-credentials `
  --resource-group $ResourceGroup `
  --name $ClusterName

az aks install-cli
kubectl get nodes
```

`az aks create` provisions a managed node resource group in addition to the
resource group above. AKS also creates/uses networking, managed identities,
disks, and load-balancer resources. Azure names and defaults can vary by
subscription policy.

## Build and push the container images

Run from the project root in Windows PowerShell. This publishes the images to
your private ACR; it does not publish source code.

```powershell
az acr login --name $AcrName

docker build -f Dockerfile.raft `
  -t "$AcrName.azurecr.io/smart-parking-raft:v1" .
docker push "$AcrName.azurecr.io/smart-parking-raft:v1"

docker build -f Dockerfile.app `
  -t "$AcrName.azurecr.io/smart-parking-web:v1" .
docker push "$AcrName.azurecr.io/smart-parking-web:v1"
```

## Install the public HTTPS ingress

The Azure DNS label on the NGINX load balancer provides a hostname in
`<label>.<region>.cloudapp.azure.com`; it avoids requiring a separately
purchased domain for this demo. Azure must accept the label as globally
available within the region. The hostname is not final until the load balancer
has an external IP.

```powershell
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update

helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx `
  --namespace ingress-nginx `
  --create-namespace `
  --set controller.service.type=LoadBalancer `
  --set "controller.service.annotations.service\.beta\.kubernetes\.io/azure-dns-label-name=$DnsLabel"

kubectl get service ingress-nginx-controller `
  --namespace ingress-nginx `
  --watch
```

Wait until `EXTERNAL-IP` is an IP address, then run:

```powershell
$ParkingHost = "$DnsLabel.$Location.cloudapp.azure.com"
Write-Output "Parking host: $ParkingHost"
```

Install cert-manager and prepare Let's Encrypt:

```powershell
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm upgrade --install cert-manager jetstack/cert-manager `
  --namespace cert-manager `
  --create-namespace `
  --set crds.enabled=true

kubectl rollout status deployment/cert-manager `
  --namespace cert-manager `
  --timeout=180s
kubectl rollout status deployment/cert-manager-webhook `
  --namespace cert-manager `
  --timeout=180s

$Email = "YOUR_REAL_EMAIL_ADDRESS"
$IssuerYaml = Get-Content .\k8s\aks\letsencrypt-issuer.yaml -Raw
$IssuerYaml = $IssuerYaml.Replace("REPLACE_WITH_EMAIL", $Email)
$IssuerYaml | kubectl apply -f -
kubectl get clusterissuer letsencrypt-prod
```

Use a real email address so certificate expiry notices can reach you. The
issuer uses ACME production; repeatedly deleting and recreating certificates
can hit Let's Encrypt rate limits.

## Deploy the app and Raft cluster with Helm

Run from the project root:

```powershell
helm lint .\helm\smart-parking

helm upgrade --install raft-node .\helm\smart-parking `
  --namespace parking `
  --create-namespace `
  --set image.repository="$AcrName.azurecr.io/smart-parking-raft" `
  --set image.tag=v1 `
  --set image.pullPolicy=IfNotPresent `
  --set parkingApp.image.repository="$AcrName.azurecr.io/smart-parking-web" `
  --set parkingApp.image.tag=v1 `
  --set parkingApp.image.pullPolicy=IfNotPresent `
  --set parkingApp.service.type=ClusterIP `
  --set parkingApp.ingress.enabled=true `
  --set parkingApp.ingress.className=nginx `
  --set parkingApp.ingress.clusterIssuer=letsencrypt-prod `
  --set parkingApp.ingress.host=$ParkingHost `
  --set parkingApp.ingress.tlsSecretName=parking-app-tls

kubectl rollout status statefulset/raft-node `
  --namespace parking `
  --timeout=300s
kubectl rollout status deployment/raft-node-parking `
  --namespace parking `
  --timeout=180s
kubectl get pods,services,ingress `
  --namespace parking `
  --output wide
```

The ingress annotation asks cert-manager to create the TLS certificate and
populate the `parking-app-tls` Secret. Allow DNS propagation and certificate
issuance, then verify:

```powershell
kubectl describe certificate parking-app-tls `
  --namespace parking
kubectl get ingress raft-node-parking `
  --namespace parking `
  --output wide
Invoke-WebRequest "https://$ParkingHost/healthz"
```

The public browser URL is:

```text
https://<DnsLabel>.<AzureRegion>.cloudapp.azure.com
```

The actual link is determined only after provisioning; do not present that
format as a live URL before the HTTPS request above succeeds.

## Parking HTTP API

- `GET /` — browser demo page
- `GET /healthz` — readiness/liveness endpoint
- `GET /api/slots/{slotId}` — returns `slotId` and `status`
- `POST /api/enter` — JSON body: `{"slotId":"slot-1","vehicleId":"vehicle-1"}`
- `POST /api/exit` — same JSON body shape

Example from PowerShell after HTTPS is working:

```powershell
$Body = @{ slotId = "slot-1"; vehicleId = "vehicle-1" } | ConvertTo-Json
Invoke-RestMethod "https://$ParkingHost/api/enter" `
  -Method Post `
  -ContentType "application/json" `
  -Body $Body
Invoke-RestMethod "https://$ParkingHost/api/slots/slot-1"
Invoke-RestMethod "https://$ParkingHost/api/exit" `
  -Method Post `
  -ContentType "application/json" `
  -Body $Body
```

## Optional Prometheus and Grafana

The Prometheus configuration discovers the Raft pod metrics port in the
`parking` namespace. Apply the RBAC before the Prometheus deployment:

```powershell
kubectl apply --namespace parking -f .\monitoring\prometheus-rbac.yaml
kubectl apply --namespace parking -f .\monitoring\prometheus-config.yaml
kubectl apply --namespace parking -f .\monitoring\prometheus-deployment.yaml
kubectl apply --namespace parking -f .\monitoring\prometheus-service.yaml
kubectl apply --namespace parking -f .\monitoring\grafana-config.yaml
kubectl apply --namespace parking -f .\monitoring\grafana-deployment.yaml
kubectl apply --namespace parking -f .\monitoring\grafana-service.yaml
```

For a private demo, use port-forwarding rather than exposing the monitoring
services publicly:

```powershell
kubectl port-forward --namespace parking service/prometheus 9090:9090
kubectl port-forward --namespace parking service/grafana 3000:3000
```

Grafana's demo credentials are configured by the existing manifest; change
them before sharing a live environment.

## Cost and resource considerations

These are planning estimates, not a quote. Confirm Azure's current regional
prices before provisioning:

- Two `Standard_D2s_v5` Linux nodes (4 vCPU and 16 GiB RAM total): often roughly
  USD $100–$200 per month if left running continuously; region, OS disks,
  discounts, and pricing changes affect the amount.
- ACR Basic: approximately USD $5/month, subject to current pricing and
  storage/transfer.
- AKS free control-plane tier: no control-plane charge, but no financially
  backed uptime SLA. Standard/Premium tiers cost more.
- The NGINX ingress `LoadBalancer` public IP, load-balancer rules, outbound
  bandwidth, and managed disks can add charges.
- Logs/monitoring retention and network egress may increase costs.

For a continuously running small demo, budget roughly USD $130–$250+ per
month as an order-of-magnitude estimate. Check the Azure Pricing Calculator
for the chosen region and subscription before creating anything.

Two `D2s_v5` nodes are a small demo cluster, not a production HA sizing
recommendation. Consider stopping/deleting the cluster when finished. Deleting
the resource group removes the demo resources, but review the target carefully
before running:

```powershell
az group delete --name $ResourceGroup --yes --no-wait
```

## Troubleshooting

```powershell
kubectl get pods --namespace parking --output wide
kubectl describe pods --namespace parking
kubectl logs deployment/raft-node-parking --namespace parking
kubectl get events --namespace parking --sort-by=.lastTimestamp
kubectl describe ingress raft-node-parking --namespace parking
kubectl describe certificate parking-app-tls --namespace parking
```
