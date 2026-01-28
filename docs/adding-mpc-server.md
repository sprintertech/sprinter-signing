# Running an MPC Signing Node

Guide for operators who have been assigned to run an MPC signing node.

## Prerequisites

- A server (VM or bare metal) with Docker and Docker Compose installed
- Network access: inbound TCP on ports `9000` (MPC p2p) and `3000` (API)
- Access to pull the Docker image `ghcr.io/sprintertech/sprinter-signing`

## What You Will Receive

Before starting, you will be provided with:

| Item | Description |
|------|-------------|
| `config.json` | Configuration file for your node |
| `SIGNING_IMAGE_VERSION` | Docker image tag to use |
| Your assigned **host port** | The host port your node should expose (e.g., `3003`) |
| Your assigned **service name** | Label for logging (e.g., `signing_relayer_4`) |

You may also receive a `keyshare` file if one has already been generated for your node. On first run before resharing, you will not have one.

## Step 1: Generate Your LibP2P Identity

Each MPC node needs a unique libp2p key pair. The private key stays with you; the peer ID is shared back with the coordinator.

```bash
docker run --rm ghcr.io/sprintertech/sprinter-signing:latest peer gen-key
```

Output:

```
LibP2P peer identity: Qm...
LibP2P private key: CAAS...
```

**Send the `peer identity` value back to the coordinator.** Keep the `private key` value -- it will be embedded into your config by the coordinator, or you will be asked to insert it yourself.

## Step 2: Set Up the Directory Structure

Create the working directory and place the files you received:

```bash
mkdir -p ~/mpc-node/cfg/keyshares
```

Copy the config file you received:

```bash
cp /path/to/config.json ~/mpc-node/cfg/config.json
```

If you received a keyshare file, copy it as well:

```bash
cp /path/to/keyshare ~/mpc-node/cfg/keyshares/0.keyshare
```

The resulting layout should be:

```
~/mpc-node/
├── docker-compose.yml
└── cfg/
    ├── config.json
    └── keyshares/
        └── 0.keyshare   (may not exist yet on first run)
```

## Step 3: Create the Docker Compose File

Create `~/mpc-node/docker-compose.yml`:

```yaml
services:
  relayer:
    image: ghcr.io/sprintertech/sprinter-signing:${SIGNING_IMAGE_VERSION}
    command: run --config /cfg/config.json
    volumes:
      - ./cfg:/cfg
    labels:
      logging: "alloy"
      logging_jobname: "containerlogs"
      service_name: "${SERVICE_NAME}"
    ports:
      - "${HOST_PORT:-3000}:3000"
    restart: always
```

## Step 4: Create the `.env` File

Create `~/mpc-node/.env`:

```bash
SIGNING_IMAGE_VERSION=<image tag provided to you>
SERVICE_NAME=<service name provided to you>
HOST_PORT=<port provided to you>
```

## Step 5: Start the Node

```bash
cd ~/mpc-node
docker compose up -d
```

Check that the container started:

```bash
docker compose ps
```

Expected output should show the container in `Up` / `running` state.

## Step 6: Wait for Resharing

If this is a new node joining an existing cluster, the coordinator will trigger a **resharing ceremony** via the admin smart contract. You do not need to do anything -- your node will automatically:

1. Detect the resharing event from the blockchain
2. Participate in the MPC resharing protocol
3. Store the resulting keyshare to `cfg/keyshares/0.keyshare`

Watch the logs to confirm resharing completes:

```bash
docker compose logs -f relayer
```

Look for log lines indicating successful resharing and keyshare storage.

## Verifying Your Node

### Check the logs

```bash
docker compose logs relayer --tail 100
```

Healthy startup logs should show:
- Successful config loading
- Topology fetched and decrypted
- P2P connections established to other peers
- Listening on the API port

### Check peer connectivity

Look in the logs for lines indicating connections to other MPC peers. All peers defined in the topology should be connected.

### Restarting

If you need to restart:

```bash
docker compose restart relayer
```

The node will reconnect to peers and resume operation. Keyshare data persists on the host in `cfg/keyshares/`.

## Updating the Node

When you receive a new image version or updated config:

1. Replace `cfg/config.json` with the new file (if config changed)
2. Update `SIGNING_IMAGE_VERSION` in `.env` (if image changed)
3. Pull the new image and recreate the container:

```bash
docker compose pull
docker compose up -d
```

## Troubleshooting

| Symptom | Likely cause | Action |
|---------|-------------|--------|
| Container exits immediately | Invalid config file | Verify `cfg/config.json` is valid JSON: `jq . ~/mpc-node/cfg/config.json` |
| `topology fetch failed` in logs | Network issue or wrong topology URL/encryption key | Verify the server can reach the topology URL in the config. Contact the coordinator. |
| No peer connections | Firewall blocking port `9000` | Ensure inbound TCP `9000` is open. Verify other peers can reach your server. |
| `keyshare not found` errors | Resharing has not happened yet | Normal for a new node. Wait for the coordinator to trigger resharing. |
| Container keeps restarting | Crash loop | Check full logs with `docker compose logs relayer` and share with the coordinator. |
