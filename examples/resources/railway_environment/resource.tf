# railway_environment creates an additional environment as a *fork* of an
# existing one. The default env (created by `railway_project`) stays as the
# empty non-fork; every additional env should be a fork of it.
#
# Under strict_env_scoping (provider default) `source_environment_id` is
# required. Passing a non-fork environment causes `serviceCreate` to leak
# across every non-fork environment in the project — this is the class of
# bug strict env-scoping prevents.

resource "railway_environment" "dev" {
  name                  = "dev"
  project_id            = railway_project.example.id
  source_environment_id = railway_project.example.default_environment.id
}
