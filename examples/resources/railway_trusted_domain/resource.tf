resource "railway_trusted_domain" "company" {
  workspace_id = var.railway_workspace_id
  domain_name  = "example.com"
  role         = "MEMBER"
}
