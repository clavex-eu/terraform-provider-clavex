# clavex_lifecycle_rule (Resource)

Manages a joiner/mover/leaver lifecycle automation rule.

## Example Usage

```terraform
resource "clavex_lifecycle_rule" "joiner_dept" {
  org_id     = var.org_id
  name       = "Engineering joiner"
  trigger    = "joiner"
  priority   = 10
  conditions = jsonencode([{ field = "department", op = "eq", value = "engineering" }])
  actions    = jsonencode([{ type = "add_to_group", group_name = "engineers" }])
  is_active  = true
}
```

## Schema

### Required

- `org_id` (String) Organisation ID. Forces replacement.
- `name` (String) Rule name.
- `trigger` (String) `joiner` | `mover` | `leaver`.
- `priority` (String) Evaluation priority (lower runs first).
- `conditions` (String) JSON array of condition objects.
- `actions` (String) JSON array of action objects.

### Optional

- `description` Description.
- `is_active` Whether the rule is active (default `true`).

### Read-Only

- `id` Rule ID.

## Import

```shell
terraform import clavex_lifecycle_rule.example <org_id>/<rule_id>
```
