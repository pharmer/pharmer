# E2E Test

## Export the Following Environment Variables:

### AWS
AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY

### Azure
AZURE_CLIENT_SECRET
AZURE_CLIENT_ID
AZURE_SUBSCRIPTION_ID
AZURE_TENANT_ID

### Linode:
LINODE_TOKEN

### DigitalOcean:
DIGITALOCEAN_TOKEN

### Vultr
VULTR_TOKEN

### Packet
PACKET_PROJECT_ID
PACKET_API_KEY

## For Google Cloud
GOOGLE_APPLICATION_CREDENTIALS=<path-to-service-account-key.json>

## Run the command for each provider

```bash
$ ginkgo -r --v --progress --trace test  -- --providers aws,azure,gce
```

