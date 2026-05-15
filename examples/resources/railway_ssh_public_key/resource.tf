resource "railway_ssh_public_key" "ci" {
  name         = "github-actions"
  public_key   = file("~/.ssh/id_ed25519.pub")
  workspace_id = var.railway_workspace_id
}
