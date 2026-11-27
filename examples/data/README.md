# Example Data Files

Sample data files for testing `kvx` features.

## Schema Example: users.json + users_schema.json

Demonstrates the `--schema` flag for customizing table column display.

### Without schema (default)

```bash
kvx examples/data/users.json -o table
```

Output shows all columns with original field names:

```text
+--------------------------- kvx ----------------------------+
|#    department   full_name    internal_code  score  user_id|
|------------------------------------------------------------|
|1    Engineering  Alice Smith  X123           95     u001   |
|2    Sales        Bob Jones    Y456           87     u002   |
|3    Engineering  Carol White  Z789           92     u003   |
|4    Marketing    David Lee    W012           78     u004   |
|5    Engineering  Eva Brown    V345           99     u005   |
+ _ ---------------------------------------------- list: 1/5 +
```

### With schema

```bash
kvx examples/data/users.json --schema examples/data/users_schema.json -o table
```

Output with schema-driven display:

```text
+--------------------------- kvx ----------------------------+
|#    Dept         Name         Score  ID                    |
|------------------------------------------                  |
|1    Engineering  Alice Smith     95  u001                  |
|2    Sales        Bob Jones       87  u002                  |
|3    Engineering  Carol White     92  u003                  |
|4    Marketing    David Lee       78  u004                  |
|5    Engineering  Eva Brown       99  u005                  |
+ _ ---------------------------------------------- list: 1/5 +
```

### What the schema does

The [users_schema.json](users_schema.json) file applies these hints:

| Field | Schema Setting | Effect |
|-------|----------------|--------|
| `user_id` | `"title": "ID"` | Column header renamed |
| `full_name` | `"title": "Name"` | Column header renamed |
| `department` | `"title": "Dept"` | Column header renamed |
| `score` | `"type": "integer"` | Right-aligned |
| `internal_code` | `"deprecated": true` | Column hidden |

### Interactive mode

```bash
kvx examples/data/users.json --schema examples/data/users_schema.json -i
```

### With column ordering

Combine `--schema` with `--column-order` or config file column ordering:

```yaml
# ~/.config/kvx/config.yaml
ui:
  formatting:
    table:
      column_order: [full_name, user_id, score, department]
```

## Other data files

- `demo_services.json` and `demo_services_schema.json`: Microservice catalog with `x-kvx-list`, `x-kvx-detail`, serialized JSON config, and synthetic `Bearer` JWT strings
- `status_device_auth.json` and `status_device_auth_schema.json`: Async status screen fixture (`x-kvx-status`) with hotkey actions and masked inline input
- `sample_cluster.json`: Large nested cluster configuration
- `sample_employees.csv`: CSV format example
- `sample_nested.json`: Nested JSON with encoded strings for auto-decode testing
- `sample.jwt`: JWT token for decoding

## Synthetic JWT Fixtures

All JWT values in `demo_services.json`, `sample_nested.json`, and `sample.jwt` are synthetic test fixtures generated solely to demonstrate `kvx` JWT detection and decoding. They are never issued by or accepted by any real service.

Specifically, the five `Bearer` tokens in `demo_services.json` are intentionally unsigned and zero-entropy so both humans and automated secret scanners can recognize them as non-credentials while `kvx` still decodes all three segments:

- JOSE header `{"alg":"none","kid":"demo-signing-key","typ":"JWT"}` (RFC 7519 `"alg":"none"` unsecured header)
- Reserved RFC 2606 example issuer `https://auth.example.com`, `"test_fixture": true`, and expired timestamp `exp: 946684800` (`2000-01-01T00:00:00Z`)
- Zero-entropy base64url placeholder third segment `AAAA` (`[0, 0, 0]`, non-empty to satisfy `kvx`'s three-segment JWT detector without a cryptographic signature)

To reproduce a `demo_services.json` token (for example, `api-gateway`) with Python's standard library:

```bash
python3 -c 'import base64, json; b64 = lambda b: base64.urlsafe_b64encode(b).rstrip(b"=").decode(); h = b64(json.dumps({"alg":"none","kid":"demo-signing-key","typ":"JWT"}, separators=(",",":")).encode()); p = b64(json.dumps({"iss":"https://auth.example.com","sub":"svc:api-gateway","role":"edge-proxy","scope":"traffic:route certs:read","exp":946684800,"test_fixture":True}, separators=(",",":")).encode()); print(f"Bearer {h}.{p}.AAAA")'
```
