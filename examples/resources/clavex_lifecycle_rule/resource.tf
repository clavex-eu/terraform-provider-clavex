resource "clavex_lifecycle_rule" "joiner_dept" {
  org_id     = var.org_id
  name       = "Engineering joiner"
  trigger    = "joiner"
  priority   = 10
  conditions = jsonencode([{ field = "department", op = "eq", value = "engineering" }])
  actions    = jsonencode([{ type = "add_to_group", group_name = "engineers" }])
  is_active  = true
}
