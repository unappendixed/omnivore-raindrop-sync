# omnivore-raindrop-sync
Automatically sync Omnivore pages to Raindrop.io

## Setup
1. Go to https://app.raindrop.io/settings/integrations, create a new app, and copy the test token (we don't need a real token since this is for personal use only)
2. Go to https://omnivore.app/settings/webhooks and add a webhook pointing to the endpoint where you will host the sync application. (e.g. https://example.com/omnivore-raindrop-sync)
3. Copy `example.env` to `.env` and fill in the RAINDROP_TOKEN variable with your test token.
4. Your Omnivore user id can either be found in the body of the webhook request or by checking the localStorage of the Omnivore app. (run `localStorage.getItem("ajs_user_traits")` in the dev console while Omnivore is open).
5. Fill in the OMNIVORE_ID environment variable with your user id.

## Building (Docker)
1. Run `docker build .` in the project directory.

## Building (without Docker)
1. Install Go toolchain
2. Run `go build main`
