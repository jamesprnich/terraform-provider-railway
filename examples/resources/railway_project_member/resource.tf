resource "railway_project_member" "alice" {
  project_id = railway_project.main.id
  user_id    = "user-12345"
  role       = "MEMBER"
}
