resource "railway_volume_backup_schedule" "example" {
  volume_instance_id = railway_volume.postgres_data.id
  kinds              = ["DAILY"]
}
