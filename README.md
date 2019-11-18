# Route42 - Manage your DNS via K8s

---

**Route42** provides simple and pluggable DNS management via Kubernetes Custom Resource Definitions.  
It can be used to scale DNS servers declaratively and can serve as a base for other higher-level APIs like LoadBalancing, CNFs or VNFs.

---

## Installation

### 1. cert-manager

To install Route42 you will need the **cert-manager** installed.  
  **cert-manager** is used to create an manage the certificate for the webhook server.  
You can install Route42 without, but you have to bring your own certificates for the webhook server.

https://docs.cert-manager.io/en/latest/getting-started/install/kubernetes.html

### 2. deploy route42-manager

This step installs the Custom Resources and the route42 controller manager, which serves as a webhook server to validate and default created `Zones` and `RecordSets`

Clone this repository and run `make deploy`, this will execute `kustomize` and apply the generated manifests via `kubectl apply -f -`.  
  Make sure to be connected to the **RIGHT** kubernetes cluster, before executing this command.

### 3. route42-agent

Deploy the CoreDNS agent for the namespaces that you want to use it in.

Default Namespace is `route42-system`, but you can change it by executing:  
`cd config/agent && kustomize edit set namespace my-fancy-namespace`

`make deploy-agent` will then deploy the agent in the same fashion as `make deploy` above.
