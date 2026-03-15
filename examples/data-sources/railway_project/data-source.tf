# Look up by name
data "railway_project" "by_name" {
  name = "my-project"
}

# Or look up by ID
data "railway_project" "by_id" {
  id = "your-project-id"
}
