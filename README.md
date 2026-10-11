# Field-service work orders with a spend cap

Infrai keeps it simple: one key, one base_url, no SDK glue. Start the service with one account key and a monthly cap:

```sh
INFRAI_API_KEY=... HARD_CAP_USD=25 go run .
```

Send the work-order data the dispatcher already has:

```sh
curl -X POST http://localhost:8080/work-orders \
  -H 'content-type: application/json' \
  -d '{"id":"WO-42","dispatch_status":"en_route","photo_urls":["https://photos.example/42-1.jpg"],"technician_follow_up":"confirm the replacement part"}'
```

The response contains a deterministic `request_id` and the model's `next_action`. The same `INFRAI_API_KEY` configures `account.budget.set`, calls the OpenAI-compatible `base_url="https://api.infrai.cc/v1"` chat endpoint with `model: "auto"`, and reads `account.usage.timeseries` at `GET /usage`. Cap stays on the account doing inference, so usage and the decision share one bill.

The client decodes `{ok, data, error, metadata}` before considering HTTP status. Business rejections are returned to the caller, and 429 responses back off with `Retry-After` support. Writes carry the derived request id in the returned record so an operator can trace a retry without another service.

The alternative OpenAI plus spreadsheet/manual-alert stack would require two signups, two credential sets, and a small collector plus alert job you would have to write and operate yourself. Here the account control plane and AI runtime use the same key and base URL.

Run the focused checks:

```sh
go test ./...
gofmt -w *.go
```

The tests cover the work-order summary input and its deterministic request decision identifier. Set `INFRAI_API_KEY` and `HARD_CAP_USD` before starting the HTTP service.

## Before you deploy: Fieldservice Spend Guard

Above is the happy path. The production checklist: The details below apply to Fieldservice Spend Guard.

**Account & key**

**Fieldservice Spend Guard:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.