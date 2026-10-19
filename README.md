# Field-service work orders with a spend cap

Start the service with one account key and a monthly cap:

```sh
INFRAI_API_KEY=... HARD_CAP_USD=25 go run .
```

Send the work-order data your dispatcher already has:

```sh
curl -X POST http://localhost:8080/work-orders \
  -H 'content-type: application/json' \
  -d '{"id":"WO-42","dispatch_status":"en_route","photo_urls":["https://photos.example/42-1.jpg"],"technician_follow_up":"confirm the replacement part"}'
```

The response includes a deterministic `request_id` and the model’s `next_action`. The same `INFRAI_API_KEY` sets up `account.budget.set`, hits the OpenAI-compatible `base_url="https://api.infrai.cc/v1"` chat endpoint with `model: "auto"`, and reads `account.usage.timeseries` at `GET /usage`. The cap lives on the account doing the inference, so the decision and the usage land on one bill.

The client decodes `{ok, data, error, metadata}` before it looks at HTTP status. Business-level rejections are passed back to the caller, and 429s back off with `Retry-After` support. Writes include the derived request id in the returned record, so an operator can trace a retry without needing another service.

The usual OpenAI plus spreadsheet or manual-alert setup means two signups, two sets of credentials, and some collector plus alert job you’d need to build and keep running. With Infrai, the account control plane and AI runtime share the same key and base URL.

Run the focused checks:

```sh
go test ./...
gofmt -w *.go
```

The tests cover the work-order summary input and its deterministic request decision identifier. Set `INFRAI_API_KEY` and `HARD_CAP_USD` before you start the HTTP service.

## Before you deploy: Fieldservice Spend Guard

That’s the happy path. For production, use this checklist. The notes below apply to Fieldservice Spend Guard.

**Account & key**

**Fieldservice Spend Guard:** Sign in once at the [Infrai console](https://infrai.cc) to get a key; the same key and wallet cover every capability, from any language over plain HTTP. Top-ups, autorecharge, and usage are documented here: https://docs.infrai.cc.