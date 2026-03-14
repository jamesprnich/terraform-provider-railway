package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestVolumeBackupScheduleResource_basic(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{
		"updateVolumeInstanceBackupSchedule":  `{"data":{"volumeInstanceBackupScheduleUpdate":true}}`,
		"getVolumeInstanceBackupSchedules": `{"data":{"volumeInstanceBackupScheduleList":[{"id":"sched-1","kind":"DAILY","cron":"0 0 * * *","name":"Daily Backup","retentionSeconds":86400}]}}`,
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_volume_backup_schedule" "test" {
  volume_instance_id = "vi-123"
  kinds              = ["DAILY"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_volume_backup_schedule.test", "id", "vi-123"),
					resource.TestCheckResourceAttr("railway_volume_backup_schedule.test", "volume_instance_id", "vi-123"),
					resource.TestCheckResourceAttr("railway_volume_backup_schedule.test", "kinds.#", "1"),
					resource.TestCheckResourceAttr("railway_volume_backup_schedule.test", "kinds.0", "DAILY"),
				),
			},
		},
	})
}

func TestVolumeBackupScheduleResource_multipleKinds(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{
		"updateVolumeInstanceBackupSchedule":  `{"data":{"volumeInstanceBackupScheduleUpdate":true}}`,
		"getVolumeInstanceBackupSchedules": `{"data":{"volumeInstanceBackupScheduleList":[{"id":"sched-1","kind":"DAILY","cron":"0 0 * * *","name":"Daily Backup","retentionSeconds":86400},{"id":"sched-2","kind":"WEEKLY","cron":"0 0 * * 0","name":"Weekly Backup","retentionSeconds":604800}]}}`,
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_volume_backup_schedule" "test" {
  volume_instance_id = "vi-123"
  kinds              = ["DAILY", "WEEKLY"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_volume_backup_schedule.test", "id", "vi-123"),
					resource.TestCheckResourceAttr("railway_volume_backup_schedule.test", "volume_instance_id", "vi-123"),
					resource.TestCheckResourceAttr("railway_volume_backup_schedule.test", "kinds.#", "2"),
					resource.TestCheckResourceAttr("railway_volume_backup_schedule.test", "kinds.0", "DAILY"),
					resource.TestCheckResourceAttr("railway_volume_backup_schedule.test", "kinds.1", "WEEKLY"),
				),
			},
		},
	})
}

func TestVolumeBackupScheduleResource_disappears(t *testing.T) {
	srv, disappear := newDisappearsMockServer(t, mockFixtures{
		"updateVolumeInstanceBackupSchedule": `{"data":{"volumeInstanceBackupScheduleUpdate":true}}`,
		"getVolumeInstanceBackupSchedules":   `{"data":{"volumeInstanceBackupScheduleList":[{"id":"sched-1","kind":"DAILY","cron":"0 0 * * *","name":"Daily Backup","retentionSeconds":86400}]}}`,
	}, "getVolumeInstanceBackupSchedules")
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_volume_backup_schedule" "test" {
  volume_instance_id = "vi-123"
  kinds              = ["DAILY"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_volume_backup_schedule.test", "id", "vi-123"),
					func(s *terraform.State) error {
						disappear()
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestVolumeBackupScheduleResource_import(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{
		"updateVolumeInstanceBackupSchedule":  `{"data":{"volumeInstanceBackupScheduleUpdate":true}}`,
		"getVolumeInstanceBackupSchedules": `{"data":{"volumeInstanceBackupScheduleList":[{"id":"sched-1","kind":"DAILY","cron":"0 0 * * *","name":"Daily Backup","retentionSeconds":86400}]}}`,
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_volume_backup_schedule" "test" {
  volume_instance_id = "vi-123"
  kinds              = ["DAILY"]
}
`,
			},
			{
				ResourceName:      "railway_volume_backup_schedule.test",
				ImportState:       true,
				ImportStateId:     "vi-123",
				ImportStateVerify: true,
			},
		},
	})
}
