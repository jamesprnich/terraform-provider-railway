# Import a fork-scoped service (strict env-scoping — provider default).
# The environment_id part is required so the imported state carries the
# fork the service belongs to; without it the next plan would see the fork
# env_id in HCL as a change requiring replace, silently destroying and
# re-creating the just-imported service.
tofu import railway_service.api your-service-id:your-environment-id

# Import a project-wide service (permissive env-scoping — provider must
# have `strict_env_scoping = false` set).
tofu import railway_service.api your-service-id
