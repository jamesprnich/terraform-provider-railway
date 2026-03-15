resource "railway_service_instance" "api_staging" {
  service_id     = railway_service.api.id
  environment_id = railway_environment.staging.id

  # Optional: override source for this environment
  # source_image = "myorg/myapp:staging"

  # Optional: build and deploy settings
  # build_command    = "npm run build"
  # start_command    = "npm start"
  # healthcheck_path = "/health"
  # num_replicas     = 1
  # region           = "us-west1"
  # vcpus            = 2
  # memory_gb        = 0.5

  # Optional: deploy behavior
  # sleep_application          = true
  # overlap_seconds            = 5
  # draining_seconds           = 10
  # healthcheck_timeout        = 300
  # restart_policy_type        = "ON_FAILURE"
  # restart_policy_max_retries = 3
  # builder                    = "RAILPACK"
  # watch_patterns             = ["backend/**"]
  # pre_deploy_command         = ["python manage.py migrate"]
}
