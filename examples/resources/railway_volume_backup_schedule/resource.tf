resource "railway_volume_backup_schedule" "daily" {
  volume_instance_id = railway_volume.data.volume_instance_id
  kinds              = ["DAILY"]

  # Valid values: "DAILY", "WEEKLY", "MONTHLY"
}
