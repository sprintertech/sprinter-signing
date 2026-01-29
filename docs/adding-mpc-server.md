# Running an MPC Signing Node

Guide for operators who have been assigned to run an MPC signing node.

## Prerequisites

- A server (VM or bare metal) with Docker and Docker Compose installed
- A domain name (preferred) or static IP address pointing to your server
- Network access: inbound TCP on ports `9000` (MPC p2p) and `3000` (API)
- Access to pull the Docker image `ghcr.io/sprintertech/sprinter-signing`

## What You Will Receive

**Before setup (Step 1):**

| Item | Description |
|------|-------------|
| `config.json` | Configuration template for your node (you will edit it) |

**After you send back your details (Step 4), the coordinator will prepare everything and give you:**

| Item | Description |
|------|-------------|
| `docker-compose.yml` | Ready-to-use Docker Compose file for your node |
| `.env` | Environment file with image version, service name, and port |

## Step 1: Generate Your LibP2P Identity

Each MPC node needs a unique libp2p key pair. You generate it and keep the private key.

```bash
docker run --rm ghcr.io/sprintertech/sprinter-signing:latest peer gen-key
```

Output:

```
LibP2P peer identity: Qm...
LibP2P private key: CAAS...
```

Save both values. You will need them in the next step.

## Step 2: Insert Your Private Key Into the Config

Open the `config.json` you received and set the `key` field under `relayer.mpcConfig` to the **LibP2P private key** from Step 1:

```json
{
  "relayer": {
    "mpcConfig": {
      "key": "<paste your LibP2P private key here>",
      ...
    },
    ...
  },
  ...
}
```

Save the file.

## Step 3: Send Your Details to the Coordinator

Send the following to the coordinator:

1. Your **LibP2P peer identity** (the `Qm...` value from Step 1)
2. The **domain name** (preferred) or **static IP** of your server

The coordinator needs both to register your node in the network topology.

## Step 4: Wait for the Go Signal

The coordinator will use your peer ID and domain to update the topology and prepare your deployment files. **Do not start the node until the coordinator gives you the green light.**

You will receive:
- `docker-compose.yml`
- `.env`

## Step 5: Set Up the Directory and Start the Node

Once you have all files and the coordinator confirms you are ready:

```bash
mkdir -p ~/mpc-node/cfg/keyshares
```

Place the files:

```bash
cp /path/to/config.json   ~/mpc-node/cfg/config.json
cp /path/to/docker-compose.yml ~/mpc-node/docker-compose.yml
cp /path/to/.env           ~/mpc-node/.env
```

Verify the layout:

```
~/mpc-node/
├── docker-compose.yml
├── .env
└── cfg/
    ├── config.json
    ├── topology        (created automatically by the node)
    └── keyshares/
```

The `cfg/` directory is mounted into the container. The node persists both the topology cache and keyshares there, so they survive container restarts and image updates.

Start the node:

```bash
cd ~/mpc-node
docker compose up -d
```

Check that the container is running:

```bash
docker compose ps
```

## Step 6: Wait for Resharing

The coordinator will trigger a **resharing ceremony** via the admin smart contract. You do not need to do anything -- your node will automatically:

1. Detect the resharing event from the blockchain
2. Participate in the MPC resharing protocol
3. Store the resulting keyshare to `cfg/keyshares/`

Watch the logs to confirm resharing completes:

```bash
docker compose logs -f relayer
```

Look for log lines indicating successful resharing and keyshare storage. Once complete, notify the coordinator.

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
2. Re-apply your private key in the config if it was overwritten
3. Update `.env` (if image version changed)
4. Pull the new image and recreate the container:

```bash
docker compose pull
docker compose up -d
```

## Troubleshooting

| Symptom | Likely cause | Action |
|---------|-------------|--------|
| Container exits immediately | Invalid config file | Verify `cfg/config.json` is valid JSON: `jq . ~/mpc-node/cfg/config.json` |
| `topology fetch failed` in logs | Topology not updated yet or network issue | Confirm with the coordinator that the topology includes your peer ID. Verify the server can reach the topology URL. |
| No peer connections | Firewall blocking port `9000` | Ensure inbound TCP `9000` is open. Verify other peers can reach your server on the domain/IP you provided. |
| `keyshare not found` errors | Resharing has not happened yet | Normal for a new node. Wait for the coordinator to trigger resharing. |
| Container keeps restarting | Crash loop | Check full logs with `docker compose logs relayer` and share with the coordinator. |
